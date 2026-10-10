package com.brebond.tipsarr

import androidx.compose.animation.Crossfade
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.rememberCoroutineScope
import com.brebond.tipsarr.core.auth.SessionManager
import com.brebond.tipsarr.core.auth.SessionPhase
import com.brebond.tipsarr.features.connect.ConnectScreen
import com.brebond.tipsarr.features.login.LoginScreen
import com.brebond.tipsarr.features.system.FailedScreen
import com.brebond.tipsarr.features.system.LaunchScreen
import com.brebond.tipsarr.features.system.OfflineScreen
import com.brebond.tipsarr.features.system.SessionExpiredScreen
import com.brebond.tipsarr.features.system.UpdateRequiredScreen
import kotlinx.coroutines.launch

/** What the app shows for the current launch-flow phase (the iOS `RootView`). */
@Composable
fun Root(session: SessionManager) {
    val phase by session.phase.collectAsState()
    val scope = rememberCoroutineScope()
    Crossfade(targetState = phase, label = "phase") { current ->
        when (current) {
            SessionPhase.Launching -> LaunchScreen()
            SessionPhase.Connect -> ConnectScreen(session)
            is SessionPhase.Login -> LoginScreen(session, current.server, current.prefillUsername)
            SessionPhase.Offline -> OfflineScreen { scope.launch { session.retry() } }
            is SessionPhase.Failed -> FailedScreen(current.error) { scope.launch { session.retry() } }
            is SessionPhase.UpdateRequired -> UpdateRequiredScreen(current.minVersion)
            is SessionPhase.SessionExpired -> SessionExpiredScreen(session, current.account, current.server)
            is SessionPhase.Ready -> MainTabs(current.account, current.profile, session.serverStatus?.userFolderChoice ?: false) { scope.launch { session.signOut(current.account) } }
        }
    }
}
