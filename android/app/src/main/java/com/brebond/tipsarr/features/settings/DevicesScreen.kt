package com.brebond.tipsarr.features.settings

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.Close
import androidx.compose.material.icons.outlined.Language
import androidx.compose.material.icons.outlined.PhoneAndroid
import androidx.compose.material.icons.outlined.PhoneIphone
import androidx.compose.material.icons.outlined.WarningAmber
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.brebond.tipsarr.R
import com.brebond.tipsarr.core.api.ApiError
import com.brebond.tipsarr.core.api.DeviceSession
import com.brebond.tipsarr.core.api.TipsarrApi
import com.brebond.tipsarr.core.api.revokeOtherSessions
import com.brebond.tipsarr.core.api.revokeSession
import com.brebond.tipsarr.core.api.sessions
import com.brebond.tipsarr.core.support.LocalToast
import com.brebond.tipsarr.core.support.ToastKind
import com.brebond.tipsarr.core.support.relativeTime
import com.brebond.tipsarr.design.Tokens
import com.brebond.tipsarr.design.palette
import com.brebond.tipsarr.ui.ButtonKind
import com.brebond.tipsarr.ui.OfflineState
import com.brebond.tipsarr.ui.StateView
import com.brebond.tipsarr.ui.TipsarrButton
import com.brebond.tipsarr.ui.TopBarScaffold
import kotlinx.coroutines.launch

/** Browsers and apps where the account is signed in. */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun DevicesScreen(api: TipsarrApi, onBack: () -> Unit) {
    val context = LocalContext.current
    val toast = LocalToast.current
    val scope = rememberCoroutineScope()
    var devices by remember { mutableStateOf<List<DeviceSession>>(emptyList()) }
    var error by remember { mutableStateOf<ApiError?>(null) }
    var loaded by remember { mutableStateOf(false) }
    var refreshing by remember { mutableStateOf(false) }
    var confirmOthers by remember { mutableStateOf(false) }

    suspend fun load() {
        try { devices = api.sessions().sortedByDescending { it.lastSeenAt }; error = null } catch (e: ApiError) { if (devices.isEmpty()) error = e }
        loaded = true
    }
    fun run(done: Int, work: suspend () -> Unit) {
        scope.launch { try { work(); toast?.show(context.getString(done)) } catch (e: ApiError) { toast?.show(e.message(context), ToastKind.Error) } }
    }
    LaunchedEffect(Unit) { load() }

    TopBarScaffold(stringResource(R.string.profile_devices_title), onBack) {
        PullToRefreshBox(isRefreshing = refreshing, onRefresh = { scope.launch { refreshing = true; load(); refreshing = false } }, modifier = Modifier.fillMaxSize()) {
            val failure = error
            when {
                failure != null -> if (failure == ApiError.Unreachable) OfflineState({ scope.launch { load() } })
                else StateView(Icons.Outlined.WarningAmber, failure.message(context), actionTitle = stringResource(R.string.common_retry), onAction = { scope.launch { load() } })
                !loaded -> Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) { CircularProgressIndicator() }
                else -> Box(Modifier.fillMaxSize().verticalScroll(rememberScrollState()).navigationBarsPadding(), contentAlignment = Alignment.TopCenter) {
                    Column(Modifier.widthIn(max = 720.dp).fillMaxWidth().padding(Tokens.Spacing.lg), verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.xl)) {
                        Text(stringResource(R.string.profile_devices_desc), fontSize = Tokens.FontSize.footnote, color = Tokens.palette.mutedFg)
                        Group("") {
                            devices.forEach { device ->
                                Row(Modifier.fillMaxWidth().padding(horizontal = Tokens.Spacing.md, vertical = Tokens.Spacing.sm), horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.md), verticalAlignment = Alignment.CenterVertically) {
                                    Icon(
                                        when (device.platform) { "web" -> Icons.Outlined.Language; "ios" -> Icons.Outlined.PhoneIphone; else -> Icons.Outlined.PhoneAndroid },
                                        null, tint = Tokens.palette.mutedFg,
                                    )
                                    Column(Modifier.weight(1f)) {
                                        Row(horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm), verticalAlignment = Alignment.CenterVertically) {
                                            Text(
                                                device.deviceName.ifEmpty { stringResource(R.string.profile_device_web) },
                                                fontSize = Tokens.FontSize.body, fontWeight = FontWeight.Medium, color = Tokens.palette.fg,
                                            )
                                            if (device.current) {
                                                Text(
                                                    stringResource(R.string.profile_device_current), fontSize = 11.sp, fontWeight = FontWeight.SemiBold, color = Tokens.Status.available,
                                                    modifier = Modifier.clip(CircleShape).background(Tokens.Status.available.copy(alpha = 0.15f)).padding(horizontal = Tokens.Spacing.sm, vertical = 2.dp),
                                                )
                                            }
                                        }
                                        Text(stringResource(R.string.profile_device_last_used, relativeTime(context, device.lastSeenAt)), fontSize = Tokens.FontSize.footnote, color = Tokens.palette.mutedFg)
                                    }
                                    if (!device.current) {
                                        IconButton({
                                            run(R.string.profile_device_signed_out) { api.revokeSession(device.id); devices = devices.filter { it.id != device.id } }
                                        }) { Icon(Icons.Outlined.Close, stringResource(R.string.profile_device_sign_out), tint = Tokens.palette.mutedFg) }
                                    }
                                }
                            }
                        }
                        if (devices.any { !it.current }) {
                            TipsarrButton(stringResource(R.string.profile_devices_sign_out_others), { confirmOthers = true }, fullWidth = true, kind = ButtonKind.Secondary)
                        }
                    }
                }
            }
        }
    }
    if (confirmOthers) {
        AlertDialog(
            onDismissRequest = { confirmOthers = false },
            title = { Text(stringResource(R.string.profile_devices_sign_out_others)) },
            confirmButton = {
                TextButton({
                    confirmOthers = false
                    run(R.string.profile_devices_signed_out_others) { api.revokeOtherSessions(); devices = devices.filter { it.current } }
                }) { Text(stringResource(R.string.profile_devices_sign_out_others), color = Tokens.palette.destructive) }
            },
            dismissButton = { TextButton({ confirmOthers = false }) { Text(stringResource(R.string.common_cancel)) } },
            containerColor = Tokens.palette.card,
        )
    }
}
