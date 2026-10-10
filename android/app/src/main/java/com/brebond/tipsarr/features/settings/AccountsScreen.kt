package com.brebond.tipsarr.features.settings

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.outlined.Logout
import androidx.compose.material.icons.outlined.AddCircleOutline
import androidx.compose.material.icons.outlined.Check
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.brebond.tipsarr.R
import com.brebond.tipsarr.core.auth.Account
import com.brebond.tipsarr.design.Tokens
import com.brebond.tipsarr.design.palette
import com.brebond.tipsarr.features.profile.AvatarView
import com.brebond.tipsarr.ui.TopBarScaffold

/** Several accounts on one phone: switch, add, sign out of one. */
@Composable
fun AccountsScreen(
    accounts: List<Account>,
    activeId: String?,
    avatarVersion: Int,
    onBack: () -> Unit,
    onSwitch: (Account) -> Unit,
    onAdd: () -> Unit,
    onSignOut: (Account) -> Unit,
) {
    var removing by remember { mutableStateOf<Account?>(null) }
    TopBarScaffold(stringResource(R.string.m_accounts_title), onBack) {
        Box(Modifier.fillMaxWidth().verticalScroll(rememberScrollState()).navigationBarsPadding(), contentAlignment = Alignment.TopCenter) {
            Column(Modifier.widthIn(max = 720.dp).fillMaxWidth().padding(Tokens.Spacing.lg), verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.xl)) {
                Text(stringResource(R.string.m_accounts_intro), fontSize = Tokens.FontSize.footnote, color = Tokens.palette.mutedFg)
                Group(stringResource(R.string.m_accounts_title)) {
                    accounts.forEach { account ->
                        Row(
                            Modifier.fillMaxWidth().clickable { onSwitch(account) }.padding(horizontal = Tokens.Spacing.md, vertical = Tokens.Spacing.sm),
                            horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.md), verticalAlignment = Alignment.CenterVertically,
                        ) {
                            AvatarView(account.userId, account.name, 40.dp, avatarVersion)
                            Column(Modifier.weight(1f)) {
                                Text(account.name, fontSize = Tokens.FontSize.body, fontWeight = FontWeight.Medium, color = Tokens.palette.fg)
                                Text(account.host, fontSize = Tokens.FontSize.footnote, color = Tokens.palette.mutedFg)
                            }
                            if (account.id == activeId) Icon(Icons.Outlined.Check, stringResource(R.string.m_accounts_active), tint = Tokens.palette.fg)
                            IconButton({ removing = account }) { Icon(Icons.AutoMirrored.Outlined.Logout, stringResource(R.string.m_accounts_remove), tint = Tokens.palette.mutedFg) }
                        }
                    }
                }
                Group("") { NavRow(Icons.Outlined.AddCircleOutline, stringResource(R.string.m_accounts_add), onClick = onAdd) }
                Text(stringResource(R.string.m_accounts_footer_signout), fontSize = Tokens.FontSize.footnote, color = Tokens.palette.mutedFg)
            }
        }
    }
    removing?.let { account ->
        AlertDialog(
            onDismissRequest = { removing = null },
            title = { Text(stringResource(R.string.m_settings_sign_out_confirm)) },
            confirmButton = { TextButton({ removing = null; onSignOut(account) }) { Text(stringResource(R.string.m_settings_sign_out), color = Tokens.palette.destructive) } },
            dismissButton = { TextButton({ removing = null }) { Text(stringResource(R.string.common_cancel)) } },
            containerColor = Tokens.palette.card,
        )
    }
}
