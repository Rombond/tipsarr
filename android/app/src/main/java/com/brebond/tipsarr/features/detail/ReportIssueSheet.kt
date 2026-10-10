package com.brebond.tipsarr.features.detail

import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.defaultMinSize
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.Flag
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
import com.brebond.tipsarr.core.api.IssueKind
import com.brebond.tipsarr.design.Tokens
import com.brebond.tipsarr.design.palette
import com.brebond.tipsarr.ui.Banner
import com.brebond.tipsarr.ui.BannerKind
import com.brebond.tipsarr.ui.ButtonKind
import com.brebond.tipsarr.ui.Chip
import com.brebond.tipsarr.ui.TipsarrButton
import com.brebond.tipsarr.ui.TipsarrSheet
import kotlinx.coroutines.launch

private fun IssueKind.title(): Int = when (this) {
    IssueKind.video -> R.string.issue_kind_video
    IssueKind.audio -> R.string.issue_kind_audio
    IssueKind.subtitles -> R.string.issue_kind_subtitles
    IssueKind.other -> R.string.issue_kind_other
}

/** Report a problem with a title: kind and a description. */
@Composable
fun ReportIssueSheet(title: String, onDismiss: () -> Unit, onSend: suspend (IssueKind, String) -> Unit) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    var kind by remember { mutableStateOf(IssueKind.video) }
    var message by remember { mutableStateOf("") }
    var sending by remember { mutableStateOf(false) }
    var errorText by remember { mutableStateOf<String?>(null) }

    fun send() {
        sending = true
        errorText = null
        scope.launch {
            try { onSend(kind, message.trim()) } catch (e: ApiError) { errorText = e.message(context) } finally { sending = false }
        }
    }

    TipsarrSheet(onDismiss, fullHeight = true) {
        Column(
            Modifier.fillMaxWidth().imePadding().verticalScroll(rememberScrollState()).navigationBarsPadding()
                .padding(horizontal = Tokens.Spacing.xl).padding(bottom = Tokens.Spacing.xl),
            verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.lg),
        ) {
            Text(stringResource(R.string.issue_report_title, title), fontSize = Tokens.FontSize.title3, fontWeight = FontWeight.SemiBold, color = Tokens.palette.fg)
            Text(stringResource(R.string.issue_report_desc), fontSize = Tokens.FontSize.subhead, color = Tokens.palette.mutedFg)
            Row(Modifier.fillMaxWidth().horizontalScroll(rememberScrollState()), horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm)) {
                IssueKind.entries.forEach { item -> Chip(stringResource(item.title()), selected = kind == item, onClick = { kind = item }) }
            }
            Column(verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.xs)) {
                Text(stringResource(R.string.issue_describe), fontSize = Tokens.FontSize.footnote, fontWeight = FontWeight.Medium, color = Tokens.palette.mutedFg)
                OutlinedTextField(
                    value = message, onValueChange = { message = it },
                    modifier = Modifier.fillMaxWidth().defaultMinSize(minHeight = 140.dp),
                    placeholder = { Text(stringResource(R.string.issue_describe_placeholder), color = Tokens.palette.mutedFg) },
                    keyboardOptions = KeyboardOptions(capitalization = KeyboardCapitalization.Sentences),
                    shape = RoundedCornerShape(Tokens.Radius.md),
                    colors = OutlinedTextFieldDefaults.colors(
                        focusedContainerColor = Tokens.palette.muted, unfocusedContainerColor = Tokens.palette.muted,
                        focusedBorderColor = Tokens.palette.ring, unfocusedBorderColor = Tokens.palette.border,
                        focusedTextColor = Tokens.palette.fg, unfocusedTextColor = Tokens.palette.fg, cursorColor = Tokens.palette.fg,
                    ),
                )
            }
            errorText?.let { Banner(it, kind = BannerKind.Error) }
            Column(verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm)) {
                TipsarrButton(
                    stringResource(if (sending) R.string.common_saving else R.string.m_issue_send), ::send,
                    fullWidth = true, enabled = !sending && message.isNotBlank(),
                ) { Icon(Icons.Outlined.Flag, null, Modifier.size(20.dp)); Spacer(Modifier.width(Tokens.Spacing.sm)) }
                TipsarrButton(stringResource(R.string.common_cancel), onDismiss, kind = ButtonKind.Ghost, fullWidth = true)
            }
        }
    }
}
