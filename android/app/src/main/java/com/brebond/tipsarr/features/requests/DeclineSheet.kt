package com.brebond.tipsarr.features.requests

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.defaultMinSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.Close
import androidx.compose.material3.Icon
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.OutlinedTextFieldDefaults
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardCapitalization
import androidx.compose.ui.unit.dp
import com.brebond.tipsarr.R
import com.brebond.tipsarr.core.api.ApiError
import com.brebond.tipsarr.design.Tokens
import com.brebond.tipsarr.design.palette
import com.brebond.tipsarr.ui.Banner
import com.brebond.tipsarr.ui.BannerKind
import com.brebond.tipsarr.ui.ButtonKind
import com.brebond.tipsarr.ui.TipsarrButton
import com.brebond.tipsarr.ui.TipsarrSheet
import kotlinx.coroutines.launch

/** Decline a request with an optional reason. */
@Composable
fun DeclineSheet(title: String, onDismiss: () -> Unit, onDecline: suspend (String) -> Unit) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    var reason by remember { mutableStateOf("") }
    var sending by remember { mutableStateOf(false) }
    var errorText by remember { mutableStateOf<String?>(null) }

    TipsarrSheet(onDismiss) {
        Column(
            Modifier.fillMaxWidth().imePadding().verticalScroll(rememberScrollState()).navigationBarsPadding()
                .padding(horizontal = Tokens.Spacing.xl).padding(bottom = Tokens.Spacing.xl),
            verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.lg),
        ) {
            Text(stringResource(R.string.requests_decline_title, title), fontSize = Tokens.FontSize.title3, fontWeight = FontWeight.SemiBold, color = Tokens.palette.fg)
            OutlinedTextField(
                value = reason, onValueChange = { reason = it },
                modifier = Modifier.fillMaxWidth().defaultMinSize(minHeight = 96.dp),
                placeholder = { Text(stringResource(R.string.requests_decline_placeholder), color = Tokens.palette.mutedFg) },
                keyboardOptions = KeyboardOptions(capitalization = KeyboardCapitalization.Sentences),
                shape = RoundedCornerShape(Tokens.Radius.md),
                colors = OutlinedTextFieldDefaults.colors(
                    focusedContainerColor = Tokens.palette.muted, unfocusedContainerColor = Tokens.palette.muted,
                    focusedBorderColor = Tokens.palette.ring, unfocusedBorderColor = Tokens.palette.border,
                    focusedTextColor = Tokens.palette.fg, unfocusedTextColor = Tokens.palette.fg, cursorColor = Tokens.palette.fg,
                ),
            )
            errorText?.let { Banner(it, kind = BannerKind.Error) }
            Column(verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm)) {
                TipsarrButton(
                    stringResource(R.string.m_requests_decline_send),
                    {
                        sending = true
                        errorText = null
                        scope.launch {
                            try { onDecline(reason.trim()) } catch (e: ApiError) { errorText = e.message(context) } finally { sending = false }
                        }
                    },
                    fullWidth = true, kind = ButtonKind.Destructive, enabled = !sending,
                ) { Icon(Icons.Outlined.Close, null, Modifier.size(20.dp)); Spacer(Modifier.width(Tokens.Spacing.sm)) }
                TipsarrButton(stringResource(R.string.common_cancel), onDismiss, fullWidth = true, kind = ButtonKind.Ghost)
            }
        }
    }
}
