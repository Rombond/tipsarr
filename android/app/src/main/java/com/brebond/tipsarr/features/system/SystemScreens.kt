package com.brebond.tipsarr.features.system

import android.content.Intent
import android.net.Uri
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.LockClock
import androidx.compose.material.icons.outlined.SystemUpdate
import androidx.compose.material.icons.outlined.WarningAmber
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import com.brebond.tipsarr.BuildConfig
import com.brebond.tipsarr.R
import com.brebond.tipsarr.core.api.ApiError
import com.brebond.tipsarr.core.api.Server
import com.brebond.tipsarr.core.auth.Account
import com.brebond.tipsarr.core.auth.SessionManager
import com.brebond.tipsarr.design.Tokens
import com.brebond.tipsarr.design.palette
import com.brebond.tipsarr.ui.ButtonKind
import com.brebond.tipsarr.ui.OfflineState
import com.brebond.tipsarr.ui.StateView
import com.brebond.tipsarr.ui.TipsarrButton

/** Shown while the launch flow checks the server. */
@Composable
fun LaunchScreen() {
    val label = stringResource(R.string.common_loading)
    Box(Modifier.fillMaxSize().background(Tokens.palette.bg), contentAlignment = Alignment.Center) {
        CircularProgressIndicator(Modifier.semantics { contentDescription = label }, color = Tokens.palette.mutedFg)
    }
}

@Composable
fun OfflineScreen(onRetry: () -> Unit) {
    Box(Modifier.fillMaxSize().background(Tokens.palette.bg)) { OfflineState(onRetry) }
}

@Composable
fun FailedScreen(error: ApiError, onRetry: () -> Unit) {
    val context = LocalContext.current
    Box(Modifier.fillMaxSize().background(Tokens.palette.bg)) {
        StateView(Icons.Outlined.WarningAmber, error.message(context), actionTitle = stringResource(R.string.common_retry), onAction = onRetry)
    }
}

/** The session ended (token revoked or expired): sign in again with the same account. */
@Composable
fun SessionExpiredScreen(session: SessionManager, account: Account, server: Server) {
    Column(Modifier.fillMaxSize().background(Tokens.palette.bg), verticalArrangement = Arrangement.Bottom) {
        Box(Modifier.weight(1f)) {
            StateView(Icons.Outlined.LockClock, stringResource(R.string.m_session_expired_title), message = stringResource(R.string.m_session_expired_body))
        }
        Box(Modifier.padding(horizontal = Tokens.Spacing.x2xl).padding(bottom = Tokens.Spacing.x3xl)) {
            TipsarrButton(stringResource(R.string.m_session_sign_in), { session.continueToLogin(server, account.name) }, fullWidth = true)
        }
    }
}

/** The server needs a newer app than this one. */
@Composable
fun UpdateRequiredScreen(minVersion: String) {
    val context = LocalContext.current
    Column(
        Modifier.fillMaxSize().background(Tokens.palette.bg),
        verticalArrangement = Arrangement.Center,
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        StateView(Icons.Outlined.SystemUpdate, stringResource(R.string.m_update_title), Modifier.weight(1f, fill = false), message = stringResource(R.string.m_update_body))
        if (minVersion.isNotEmpty()) Text(minVersion, fontSize = Tokens.FontSize.footnote, color = Tokens.palette.mutedFg)
        TipsarrButton(
            stringResource(R.string.m_update_open_store),
            { context.startActivity(Intent(Intent.ACTION_VIEW, Uri.parse("market://details?id=${BuildConfig.APPLICATION_ID}")).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)) },
            Modifier.padding(Tokens.Spacing.x2xl),
        )
    }
}
