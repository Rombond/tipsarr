package com.brebond.tipsarr.features.connect

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.size
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.Dns
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import com.brebond.tipsarr.R
import com.brebond.tipsarr.core.api.ApiError
import com.brebond.tipsarr.core.auth.ServerAddress
import com.brebond.tipsarr.core.auth.SessionManager
import com.brebond.tipsarr.design.Tokens
import com.brebond.tipsarr.design.palette
import com.brebond.tipsarr.features.system.AuthScaffold
import com.brebond.tipsarr.ui.ButtonKind
import com.brebond.tipsarr.ui.Field
import com.brebond.tipsarr.ui.TipsarrButton
import kotlinx.coroutines.launch

/** First screen: the server address. Continues only when it answers `/status` like a Tipsarr server. */
@Composable
fun ConnectScreen(session: SessionManager) {
    var address by remember { mutableStateOf("") }
    var checking by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<Int?>(null) }
    val scope = rememberCoroutineScope()

    fun submit() {
        if (checking) return
        val url = ServerAddress.normalize(address)
        if (url == null) {
            error = R.string.error_url_invalid
            return
        }
        checking = true
        scope.launch {
            try {
                session.continueToLogin(session.connect(url))
            } catch (e: ApiError) {
                // Reachable but not ours (404, HTML, bad JSON) vs not reachable at all.
                error = if (e == ApiError.Unreachable) R.string.m_connect_unreachable else R.string.m_connect_not_tipsarr
            } finally {
                checking = false
            }
        }
    }

    AuthScaffold(Icons.Outlined.Dns, stringResource(R.string.m_connect_title), stringResource(R.string.m_connect_subtitle)) {
        Column(verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.lg)) {
            Field(
                title = stringResource(R.string.m_connect_address),
                value = address,
                onValueChange = { address = it; error = null },
                placeholder = stringResource(R.string.m_connect_placeholder),
                error = error?.let { stringResource(it) },
                keyboardType = KeyboardType.Uri,
                imeAction = ImeAction.Go,
                onSubmit = ::submit,
            )
            if (checking) {
                TipsarrButton(stringResource(R.string.m_connect_checking), {}, fullWidth = true, enabled = false) {
                    CircularProgressIndicator(Modifier.size(18.dp), strokeWidth = 2.dp, color = Tokens.palette.primaryFg)
                    androidx.compose.foundation.layout.Spacer(Modifier.size(Tokens.Spacing.sm))
                }
            } else {
                TipsarrButton(stringResource(R.string.m_connect_continue), ::submit, fullWidth = true, enabled = address.isNotBlank())
            }
            if (session.accounts.active != null) {
                TipsarrButton(stringResource(R.string.common_cancel), { scope.launch { session.cancelAddAccount() } }, kind = ButtonKind.Ghost, fullWidth = true)
            }
            Text(stringResource(R.string.m_connect_footer), fontSize = Tokens.FontSize.footnote, color = Tokens.palette.mutedFg)
        }
    }
}
