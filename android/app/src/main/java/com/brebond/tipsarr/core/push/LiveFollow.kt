package com.brebond.tipsarr.core.push

import android.Manifest
import android.app.Application
import android.content.Context
import android.content.pm.PackageManager
import android.os.Build
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.core.content.ContextCompat
import com.brebond.tipsarr.core.api.RequestFollow
import com.brebond.tipsarr.core.api.TipsarrApi
import com.brebond.tipsarr.core.api.follow
import com.brebond.tipsarr.core.api.follows
import com.brebond.tipsarr.core.api.unfollow
import com.brebond.tipsarr.core.auth.Account
import com.brebond.tipsarr.core.auth.AccountStore
import com.google.firebase.messaging.FirebaseMessaging
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import kotlinx.coroutines.suspendCancellableCoroutine
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json
import kotlin.coroutines.resume

/**
 * Follow a request outside the app (issue #73): the server pushes data-only `live` messages and the app keeps one
 * ongoing notification per followed request (a Live Update on Android 16+, a progress notification before).
 * The local cache holds what the push does not carry (title, account) and the last shown time, to drop stale pushes.
 */
object LiveFollow {
    enum class Result { Started, NotificationsOff }

    @Serializable
    data class Entry(
        val accountId: String,
        val server: String,
        val title: String,
        val mediaType: String,
        val lastUpdatedAt: Long = 0,
        val expiresAt: Long = 0,
    )

    private const val PREFS = "live_follows"
    private const val SETTINGS = "live_follow_settings"
    private const val ENABLED = "enabled"
    private val json = Json { ignoreUnknownKeys = true }
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Main.immediate)
    private lateinit var app: Application

    private val _followed = MutableStateFlow<Set<String>>(emptySet())
    /** Ids of the requests followed on this device. */
    val followed: StateFlow<Set<String>> = _followed.asStateFlow()

    /** Settings switch, on by default. Turning it off stops every follow. */
    var enabled by mutableStateOf(true)
        private set

    /** The active server offers live follow (`features.liveFollow`). */
    var available by mutableStateOf(false)
        private set

    fun init(application: Application) {
        app = application
        enabled = application.getSharedPreferences(SETTINGS, Context.MODE_PRIVATE).getBoolean(ENABLED, true)
        _followed.value = entries(application).keys
    }

    // Cache

    private fun prefs(context: Context) = context.applicationContext.getSharedPreferences(PREFS, Context.MODE_PRIVATE)

    private fun entries(context: Context): Map<String, Entry> =
        prefs(context).all.mapNotNull { (id, value) ->
            (value as? String)?.let { runCatching { json.decodeFromString(Entry.serializer(), it) }.getOrNull() }?.let { id to it }
        }.toMap()

    fun entry(context: Context, id: String): Entry? =
        prefs(context).getString(id, null)?.let { runCatching { json.decodeFromString(Entry.serializer(), it) }.getOrNull() }

    private fun save(context: Context, id: String, entry: Entry) {
        prefs(context).edit().putString(id, json.encodeToString(Entry.serializer(), entry)).apply()
        _followed.value = _followed.value + id
    }

    private fun remove(context: Context, id: String) {
        prefs(context).edit().remove(id).apply()
        _followed.value = _followed.value - id
    }

    // Lifecycle

    /** An account is ready: reconcile the local follows with the server's (`GET /me/follows`). */
    suspend fun attach(account: Account, liveFollow: Boolean) {
        available = liveFollow
        val local = entries(app).filterValues { it.accountId == account.id }
        if (!liveFollow) { local.keys.forEach { end(app, it) }; return }
        val remote = runCatching { TipsarrApi(account.serverUrl, account.token).follows().items }.getOrNull() ?: return
        val remoteIds = remote.map { it.requestId }.toSet()
        local.keys.filter { it !in remoteIds }.forEach { end(app, it) }
        remote.forEach { follow -> local[follow.requestId]?.let { show(app, follow.requestId, it, follow) } }
    }

    /** Before sign-out: end this account's follows. */
    suspend fun detach(account: Account) {
        entries(app).filterValues { it.accountId == account.id }.keys.forEach { id ->
            runCatching { TipsarrApi(account.serverUrl, account.token).unfollow(id) }
            end(app, id)
        }
    }

    /** FCM gave a new token: every follow is sent again so the server pushes to the new one. */
    fun onNewToken(token: String) {
        scope.launch {
            val accounts = AccountStore(app).accounts.associateBy { it.id }
            entries(app).forEach { (id, entry) ->
                val account = accounts[entry.accountId] ?: return@forEach
                runCatching { TipsarrApi(account.serverUrl, account.token).follow(id, token, entry.server) }
            }
        }
    }

    fun updateEnabled(value: Boolean) {
        enabled = value
        app.getSharedPreferences(SETTINGS, Context.MODE_PRIVATE).edit().putBoolean(ENABLED, value).apply()
        if (!value) scope.launch { unfollowAll() }
    }

    private suspend fun unfollowAll() {
        val accounts = AccountStore(app).accounts.associateBy { it.id }
        entries(app).forEach { (id, entry) ->
            accounts[entry.accountId]?.let { runCatching { TipsarrApi(it.serverUrl, it.token).unfollow(id) } }
            end(app, id)
        }
    }

    // Follow

    /** Throws `ApiError` from the server (409 follow_final, follow_limit, push_unavailable...). */
    suspend fun follow(account: Account, id: String, title: String, mediaType: String): Result {
        if (!notificationsAllowed()) {
            runCatching { PushManager.permissionRequester?.invoke() }
            if (!notificationsAllowed()) return Result.NotificationsOff
        }
        val token = fetchToken() ?: throw com.brebond.tipsarr.core.api.ApiError.Unreachable
        val server = PushManager.serverKey(account.serverUrl)
        val state = TipsarrApi(account.serverUrl, account.token).follow(id, token, server)
        val entry = Entry(account.id, server, title, mediaType)
        save(app, id, entry)
        show(app, id, entry, state)
        return Result.Started
    }

    suspend fun unfollow(account: Account, id: String) {
        end(app, id)
        runCatching { TipsarrApi(account.serverUrl, account.token).unfollow(id) }
    }

    private fun show(context: Context, id: String, entry: Entry, follow: RequestFollow) {
        val state = LiveState(id, entry.server, follow.stage, follow.percent.coerceIn(0, 100), follow.etaSeconds.coerceAtLeast(0), follow.updatedAt, end = false, dismissAfter = 0)
        if (state.isStale(entry.lastUpdatedAt.takeIf { it > 0 })) return
        val updated = entry.copy(lastUpdatedAt = follow.updatedAt, expiresAt = follow.expiresAt)
        save(context, id, updated)
        LiveFollowNotification.post(context, state, updated)
    }

    // Push

    /** A `live` push arrived (service thread, no network). Unknown ids and stale messages are dropped. */
    fun onPush(context: Context, state: LiveState) {
        val entry = entry(context, state.id) ?: return
        if (state.isStale(entry.lastUpdatedAt.takeIf { it > 0 })) return
        if (state.end) {
            remove(context, state.id)
            if (state.dismissAfter <= 0) LiveFollowNotification.cancel(context, state.id)
            else LiveFollowNotification.post(context, state, entry, final = true)
            return
        }
        val updated = entry.copy(lastUpdatedAt = state.updatedAt)
        save(context, state.id, updated)
        LiveFollowNotification.post(context, state, updated)
    }

    /** Ends a follow locally: cache and notification. */
    private fun end(context: Context, id: String) {
        remove(context, id)
        LiveFollowNotification.cancel(context, id)
    }

    private fun notificationsAllowed() = Build.VERSION.SDK_INT < 33 ||
        ContextCompat.checkSelfPermission(app, Manifest.permission.POST_NOTIFICATIONS) == PackageManager.PERMISSION_GRANTED

    private suspend fun fetchToken(): String? = suspendCancellableCoroutine { cont ->
        FirebaseMessaging.getInstance().token.addOnCompleteListener { if (cont.isActive) cont.resume(if (it.isSuccessful) it.result else null) }
    }
}
