package com.brebond.tipsarr.ui

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.Search
import androidx.compose.material.icons.outlined.WifiOff
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.unit.dp
import com.brebond.tipsarr.R
import com.brebond.tipsarr.design.Tokens
import com.brebond.tipsarr.design.palette

/** Full-area empty, error or offline state: symbol, title, optional message and one action. */
@Composable
fun StateView(
    icon: ImageVector,
    title: String,
    modifier: Modifier = Modifier,
    message: String? = null,
    actionTitle: String? = null,
    onAction: (() -> Unit)? = null,
) {
    Column(
        modifier.fillMaxSize().padding(Tokens.Spacing.x3xl),
        verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.lg, Alignment.CenterVertically),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Icon(icon, contentDescription = null, tint = Tokens.palette.mutedFg, modifier = Modifier.size(40.dp))
        Column(verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm), horizontalAlignment = Alignment.CenterHorizontally) {
            Text(title, fontSize = Tokens.FontSize.title3, fontWeight = FontWeight.SemiBold, color = Tokens.palette.fg, textAlign = TextAlign.Center)
            if (message != null) Text(message, fontSize = Tokens.FontSize.subhead, color = Tokens.palette.mutedFg, textAlign = TextAlign.Center)
        }
        if (actionTitle != null && onAction != null) TipsarrButton(actionTitle, onAction)
    }
}

/** "No connection" with a retry button. */
@Composable
fun OfflineState(onRetry: () -> Unit, modifier: Modifier = Modifier) = StateView(
    icon = Icons.Outlined.WifiOff,
    title = stringResource(R.string.m_offline_title),
    message = stringResource(R.string.m_offline_body),
    actionTitle = stringResource(R.string.m_offline_retry),
    onAction = onRetry,
    modifier = modifier,
)

@Composable
fun NoResultsState(modifier: Modifier = Modifier) = StateView(
    icon = Icons.Outlined.Search,
    title = stringResource(R.string.common_no_results),
    modifier = modifier,
)
