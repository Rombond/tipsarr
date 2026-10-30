package com.brebond.tipsarr.core.push

import android.Manifest
import android.app.Application
import android.app.NotificationChannel
import android.app.NotificationManager
import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.net.Uri
import android.os.Build
import android.provider.Settings
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.core.app.NotificationManagerCompat
import androidx.core.content.ContextCompat
import com.brebond.tipsarr.R
import com.brebond.tipsarr.core.api.TipsarrApi
import com.brebond.tipsarr.core.api.registerDevice
import com.brebond.tipsarr.core.api.unregisterDevice
import com.brebond.tipsarr.core.auth.Account
import com.brebond.tipsarr.core.auth.AccountStore
import com.google.firebase.messaging.FirebaseMessaging
import kotlinx.coroutines.CompletableDeferred
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.launch
import kotlinx.coroutines.suspendCancellableCoroutine
import java.net.URI
import java.util.Locale
import kotlin.coroutines.resume

/**
 * Push registration of this install (the iOS `PushManager`): asks for the notification permission, gets the FCM token,
 * tells the active server about it (`PUT /me/devices/current`) and keeps the notification toggles. The server only
 * receives a token; alerts carry an event and an id, the app fetches the details when opened.
 */
object PushManager {
    /** Notification categories, a bitmask shared with the server. Also the notification channel ids. */
    const val REQUESTS = 1
    const val ADMIN = 2
    const val ISSUES = 4
    private const val ALL = REQUESTS or ADMIN or ISSUES

    private const val PREFS = "push"
    private const val ASKED = "askedForPermission"

    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Main.immediate)
    private lateinit var app: Application

    /** Set by the activity: shows the system permission dialog and returns the answer. */
    var permissionRequester: (suspend () -> Boolean)? = null

    var authorized by mutableStateOf(false)
        private set
    /** The server knows this device; `categories` is its copy of the toggles. */
    var registered by mutableStateOf(false)
        private set
    var categories by mutableIntStateOf(ALL)
        private set

    private var account: Account? = null
    private var serverAllowsPush = false
    private val prefs get() = app.getSharedPreferences(PREFS, Context.MODE_PRIVATE)

    fun init(application: Application) {
        app = application
        refresh()
        createChannels(application)
    }

    private fun optOutKey(account: Account) = "optOut.${account.id}"
    private fun optedOut(account: Account) = prefs.getBoolean(optOutKey(account), false)
    private val asked get() = prefs.getBoolean(ASKED, false)

    /** True when this device is registered and allowed to show alerts. */
    val isOn: Boolean get() = registered && authorized
    /** Alerts were refused (or turned off in the system settings): only the system settings can change it. */
    val isDenied: Boolean get() = !authorized && (Build.VERSION.SDK_INT < 33 || asked)

    fun refresh() {
        authorized = NotificationManagerCompat.from(app).areNotificationsEnabled()
    }

    // Lifecycle

    /** An account is signed in and ready. Registers silently when alerts are already allowed; managers are asked at once. */
    suspend fun attach(account: Account, pushAvailable: Boolean, askNow: Boolean) {
        this.account = account
        serverAllowsPush = pushAvailable
        registered = false
        refresh()
        if (pushAvailable && askNow && !authorized && !asked && !optedOut(account)) askPermission()
        if (!pushAvailable || !authorized || optedOut(account)) return
        syncRegistration()
    }

    /** Before sign-out: stop alerts for this account on this device. */
    suspend fun detach(account: Account) {
        runCatching { TipsarrApi(account.serverUrl, account.token).unregisterDevice() }
        if (this.account?.id == account.id) registered = false
    }

    /** FCM gave a new token: every account of this phone tells its server. */
    fun onNewToken(token: String) {
        scope.launch {
            refresh()
            if (!authorized) return@launch
            AccountStore(app).accounts.filter { !optedOut(it) }.forEach { runCatching { register(it, token, null) } }
        }
    }

    // Asking

    /** First request created: the right moment to ask, once. */
    suspend fun askAfterFirstRequest() {
        if (!serverAllowsPush || asked) return
        refresh()
        if (authorized) return
        turnOn()
    }

    fun askAfterFirstRequestLater() { scope.launch { askAfterFirstRequest() } }

    /** Settings switch: asks for permission when needed and registers. */
    suspend fun turnOn() {
        val account = account ?: return
        prefs.edit().putBoolean(optOutKey(account), false).apply()
        refresh()
        if (!authorized && !asked) askPermission()
        if (!authorized) return
        syncRegistration()
    }

    suspend fun turnOff() {
        val account = account ?: return
        prefs.edit().putBoolean(optOutKey(account), true).apply()
        runCatching { TipsarrApi(account.serverUrl, account.token).unregisterDevice() }
        registered = false
    }

    fun openSystemSettings(context: Context) {
        val intent = Intent(Settings.ACTION_APP_NOTIFICATION_SETTINGS)
            .putExtra(Settings.EXTRA_APP_PACKAGE, context.packageName)
            .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
        runCatching { context.startActivity(intent) }.onFailure {
            context.startActivity(Intent(Settings.ACTION_APPLICATION_DETAILS_SETTINGS, Uri.parse("package:${context.packageName}")).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK))
        }
    }

    suspend fun set(category: Int, on: Boolean) {
        val value = if (on) categories or category else categories and category.inv()
        if (value == categories) return
        val before = categories
        categories = value
        try {
            registerActive(value)
        } catch (e: Exception) {
            categories = before
        }
    }

    // Permission and server

    private suspend fun askPermission() {
        prefs.edit().putBoolean(ASKED, true).apply()
        if (Build.VERSION.SDK_INT >= 33 &&
            ContextCompat.checkSelfPermission(app, Manifest.permission.POST_NOTIFICATIONS) != PackageManager.PERMISSION_GRANTED
        ) {
            runCatching { permissionRequester?.invoke() }
        }
        refresh()
    }

    private suspend fun fetchToken(): String? = suspendCancellableCoroutine { cont ->
        FirebaseMessaging.getInstance().token.addOnCompleteListener { if (cont.isActive) cont.resume(if (it.isSuccessful) it.result else null) }
    }

    /** Sends the token (and language) to the server; keeps the toggles the server already has. */
    private suspend fun syncRegistration() {
        val account = account ?: return
        if (!serverAllowsPush || !authorized || optedOut(account)) return
        runCatching { registerActive(null) }
    }

    private suspend fun registerActive(wanted: Int?) {
        val account = account ?: return
        val token = fetchToken() ?: return
        val result = register(account, token, wanted)
        categories = result.categories
        registered = true
    }

    private suspend fun register(account: Account, token: String, wanted: Int?) =
        TipsarrApi(account.serverUrl, account.token).registerDevice(token, Locale.getDefault().language, serverKey(account.serverUrl), wanted)

    /** `host` or `host:port`, how the alert names a server. */
    fun serverKey(serverUrl: String): String {
        val uri = runCatching { URI(serverUrl) }.getOrNull() ?: return serverUrl.lowercase()
        return (uri.host ?: "").lowercase() + (if (uri.port > 0) ":${uri.port}" else "")
    }

    // Channels

    /** One channel per category; the relay names the channel in each alert. Names follow the app language. */
    fun createChannels(context: Context) {
        if (Build.VERSION.SDK_INT < 26) return
        val manager = context.getSystemService(NotificationManager::class.java)
        listOf(
            "requests" to R.string.m_settings_notifications_requests,
            "admin" to R.string.m_settings_notifications_admin,
            "issues" to R.string.m_settings_notifications_issues,
        ).forEach { (id, name) ->
            manager.createNotificationChannel(NotificationChannel(id, context.getString(name), NotificationManager.IMPORTANCE_DEFAULT))
        }
    }

    /** The channel an event posts on (the same mapping as the relay's templates). */
    fun channel(event: String): String = when {
        event == "request.created" -> "admin"
        event.startsWith("issue.") -> "issues"
        else -> "requests"
    }
}
