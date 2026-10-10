package com.brebond.tipsarr.features.login

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.size
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.AccountCircle
import androidx.compose.material.icons.outlined.Dns
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.unit.dp
import com.brebond.tipsarr.R
import com.brebond.tipsarr.core.api.ApiError
import com.brebond.tipsarr.core.api.Server
import com.brebond.tipsarr.core.auth.SessionManager
import com.brebond.tipsarr.design.Tokens
import com.brebond.tipsarr.design.palette
import com.brebond.tipsarr.features.system.AuthScaffold
import com.brebond.tipsarr.ui.Banner
import com.brebond.tipsarr.ui.BannerKind
import com.brebond.tipsarr.ui.ButtonKind
import com.brebond.tipsarr.ui.Field
import com.brebond.tipsarr.ui.TipsarrButton
import kotlinx.coroutines.launch

/** Classic Jellyfin login; the server answers with a long-lived Bearer token. */
@Composable
fun LoginScreen(session: SessionManager, server: Server, prefillUsername: String?) {
    val context = LocalContext.current
    var username by remember { mutableStateOf(prefillUsername.orEmpty()) }
    var password by remember { mutableStateOf("") }
    var signingIn by remember { mutableStateOf(false) }
    var errorText by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    fun submit() {
        if (signingIn || username.isEmpty() || password.isEmpty()) return
        signingIn = true
        errorText = null
        scope.launch {
            try {
                session.signIn(server, username, password)
            } catch (e: ApiError) {
                password = ""
                errorText = e.message(context)
            } finally {
                signingIn = false
            }
        }
    }

    AuthScaffold(Icons.Outlined.AccountCircle, stringResource(R.string.login_title), stringResource(R.string.m_login_subtitle)) {
        Column(verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.lg)) {
            Row(horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm), verticalAlignment = Alignment.CenterVertically) {
                Icon(Icons.Outlined.Dns, contentDescription = null, tint = Tokens.palette.mutedFg, modifier = Modifier.size(16.dp))
                Text(server.host, fontSize = Tokens.FontSize.footnote, color = Tokens.palette.mutedFg)
            }
            Field(stringResource(R.string.login_username), username, { username = it }, imeAction = ImeAction.Next)
            Field(stringResource(R.string.login_password), password, { password = it }, secure = true, imeAction = ImeAction.Go, onSubmit = ::submit)
            if (errorText != null) Banner(errorText!!, kind = BannerKind.Error)
            if (signingIn) {
                TipsarrButton(stringResource(R.string.m_login_signing_in), {}, fullWidth = true, enabled = false) {
                    CircularProgressIndicator(Modifier.size(18.dp), strokeWidth = 2.dp, color = Tokens.palette.primaryFg)
                    Spacer(Modifier.size(Tokens.Spacing.sm))
                }
            } else {
                TipsarrButton(stringResource(R.string.login_submit), ::submit, fullWidth = true, enabled = username.isNotEmpty() && password.isNotEmpty())
            }
            Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                TipsarrButton(stringResource(R.string.m_accounts_other_server), session::backToConnect, kind = ButtonKind.Ghost)
                if (session.accounts.active != null) {
                    TipsarrButton(stringResource(R.string.common_cancel), { scope.launch { session.cancelAddAccount() } }, kind = ButtonKind.Ghost)
                }
            }
        }
    }
}
