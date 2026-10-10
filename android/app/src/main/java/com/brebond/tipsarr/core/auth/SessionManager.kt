package com.brebond.tipsarr.core.auth

import android.os.Build
import com.brebond.tipsarr.BuildConfig
import com.brebond.tipsarr.core.api.ApiError
import com.brebond.tipsarr.core.api.Profile
import com.brebond.tipsarr.core.api.Server
import com.brebond.tipsarr.core.api.ServerStatus
import com.brebond.tipsarr.core.api.TipsarrApi
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow

/** What the root screen shows. Launch flow of the behaviour spec, section 1. */
sealed interface SessionPhase {
    data object Launching : SessionPhase
    data object Connect : SessionPhase
    data class Login(val server: Server, val prefillUsername: String? = null) : SessionPhase
    data object Offline : SessionPhase
    data class Failed(val error: ApiError) : SessionPhase
    data class UpdateRequired(val minVersion: String) : SessionPhase
    data class SessionExpired(val account: Account, val server: Server) : SessionPhase
    data class Ready(val account: Account, val profile: Profile) : SessionPhase
}

class SessionManager(val accounts: AccountStore) {
    private val _phase = MutableStateFlow<SessionPhase>(SessionPhase.Launching)
    val phase: StateFlow<SessionPhase> = _phase.asStateFlow()

    /** Status of the active server, read at launch or sign-in. */
    var serverStatus: ServerStatus? = null
        private set

    /** Launch: resume the active account, or ask for a server. */
    suspend fun start() {
        val account = accounts.active
        if (account == null) {
            _phase.value = SessionPhase.Connect
            return
        }
        resume(account)
    }

    suspend fun retry() {
        _phase.value = SessionPhase.Launching
        start()
    }

    private suspend fun resume(account: Account) {
        _phase.value = SessionPhase.Launching
        val server = try {
            Server(account.serverUrl, TipsarrApi(account.serverUrl).status())
        } catch (e: ApiError) {
            return fail(e)
        }
        if (AppVersion.isOlder(BuildConfig.APP_VERSION, server.status.minAppVersion)) {
            _phase.value = SessionPhase.UpdateRequired(server.status.minAppVersion)
            return
        }
        try {
            val profile = TipsarrApi(account.serverUrl, account.token).me()
            serverStatus = server.status
            accounts.add(account)
            _phase.value = SessionPhase.Ready(account, profile)
        } catch (e: ApiError) {
            if (e.isUnauthorized) _phase.value = SessionPhase.SessionExpired(account, server) else fail(e)
        }
    }

    private fun fail(error: ApiError) {
        _phase.value = if (error == ApiError.Unreachable) SessionPhase.Offline else SessionPhase.Failed(error)
    }

    /** Connect screen: the address answers `/status` like a Tipsarr server. */
    suspend fun connect(url: String): Server = Server(url, TipsarrApi(url).status())

    fun continueToLogin(server: Server, username: String? = null) {
        _phase.value = if (AppVersion.isOlder(BuildConfig.APP_VERSION, server.status.minAppVersion)) {
            SessionPhase.UpdateRequired(server.status.minAppVersion)
        } else {
            SessionPhase.Login(server, username)
        }
    }

    suspend fun signIn(server: Server, username: String, password: String) {
        val result = TipsarrApi(server.url).signIn(username, password, deviceName())
        val account = Account(server.url, result.profile.id, result.profile.name, result.token, System.currentTimeMillis())
        accounts.add(account)
        serverStatus = server.status
        _phase.value = SessionPhase.Ready(account, result.profile)
    }

    /** Removes the account from the phone; the next one (or Connect) takes over. */
    suspend fun signOut(account: Account) {
        TipsarrApi(account.serverUrl, account.token).logout()
        accounts.remove(account)
        start()
    }

    fun backToConnect() { _phase.value = SessionPhase.Connect }

    /** Switches to another stored account (no password asked). */
    suspend fun switchTo(account: Account) {
        if (account.id == accounts.activeId) return
        accounts.setActive(account.id)
        resume(account)
    }

    /** Add account: same server first (login), the login screen offers another server. */
    fun beginAddAccount() {
        val account = accounts.active
        val status = serverStatus
        if (account == null || status == null) return backToConnect()
        _phase.value = SessionPhase.Login(Server(account.serverUrl, status))
    }

    /** Leaves the add-account flow without signing in. */
    suspend fun cancelAddAccount() = start()

    /** Profile changed on the server (language, score source): keep the signed-in screens in sync. */
    fun update(profile: Profile) {
        (_phase.value as? SessionPhase.Ready)?.let { _phase.value = it.copy(profile = profile) }
    }

    private fun deviceName(): String =
        listOf(Build.MANUFACTURER, Build.MODEL).filter { it.isNotBlank() }.joinToString(" ").take(64)
}
