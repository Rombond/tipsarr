package com.brebond.tipsarr

import androidx.compose.animation.Crossfade
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.safeDrawingPadding
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import com.brebond.tipsarr.core.auth.SessionManager
import com.brebond.tipsarr.core.auth.SessionPhase
import com.brebond.tipsarr.design.Tokens
import com.brebond.tipsarr.design.palette
import com.brebond.tipsarr.features.connect.ConnectScreen
import com.brebond.tipsarr.features.login.LoginScreen
import com.brebond.tipsarr.features.system.FailedScreen
import com.brebond.tipsarr.features.system.LaunchScreen
import com.brebond.tipsarr.features.system.OfflineScreen
import com.brebond.tipsarr.features.system.SessionExpiredScreen
import com.brebond.tipsarr.features.system.UpdateRequiredScreen
import com.brebond.tipsarr.ui.ButtonKind
import com.brebond.tipsarr.ui.TipsarrButton
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
            is SessionPhase.Ready -> ReadyPlaceholder(current, onSignOut = { scope.launch { session.signOut(current.account) } })
        }
    }
}

/** Stands in for the tabs until Discover exists (Android step 3). */
@Composable
private fun ReadyPlaceholder(ready: SessionPhase.Ready, onSignOut: () -> Unit) {
    Column(
        Modifier.fillMaxSize().background(Tokens.palette.bg).safeDrawingPadding().padding(Tokens.Spacing.x2xl),
        verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.lg, androidx.compose.ui.Alignment.CenterVertically),
    ) {
        Text(ready.profile.name, fontSize = Tokens.FontSize.largeTitle, color = Tokens.palette.fg)
        Text(ready.account.host, fontSize = Tokens.FontSize.subhead, color = Tokens.palette.mutedFg)
        TipsarrButton(stringResource(R.string.m_settings_sign_out), onSignOut, kind = ButtonKind.Secondary)
    }
}
