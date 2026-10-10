package com.brebond.tipsarr.ui

import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.Close
import androidx.compose.material.icons.outlined.ErrorOutline
import androidx.compose.material.icons.outlined.Info
import androidx.compose.material.icons.outlined.WarningAmber
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.brebond.tipsarr.R
import com.brebond.tipsarr.design.Tokens
import com.brebond.tipsarr.design.palette

enum class BannerKind { Info, Warning, Error }

/** Inline notice: info, warning or error, with an optional message and dismiss button. */
@Composable
fun Banner(
    title: String,
    modifier: Modifier = Modifier,
    kind: BannerKind = BannerKind.Info,
    message: String? = null,
    onDismiss: (() -> Unit)? = null,
) {
    val color: Color = when (kind) {
        BannerKind.Info -> Tokens.Status.approved
        BannerKind.Warning -> Tokens.Status.requested
        BannerKind.Error -> Tokens.palette.destructive
    }
    val icon: ImageVector = when (kind) {
        BannerKind.Info -> Icons.Outlined.Info
        BannerKind.Warning -> Icons.Outlined.WarningAmber
        BannerKind.Error -> Icons.Outlined.ErrorOutline
    }
    val shape = RoundedCornerShape(Tokens.Radius.md)
    Row(
        modifier
            .fillMaxWidth()
            .clip(shape)
            .background(color.copy(alpha = 0.12f))
            .border(BorderStroke(1.dp, color.copy(alpha = 0.4f)), shape)
            .padding(Tokens.Spacing.md)
            .semantics(mergeDescendants = true) {},
        horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.md),
        verticalAlignment = Alignment.Top,
    ) {
        Icon(icon, contentDescription = null, tint = color)
        Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.xs)) {
            Text(title, fontSize = Tokens.FontSize.subhead, fontWeight = FontWeight.SemiBold, color = Tokens.palette.fg)
            if (message != null) Text(message, fontSize = Tokens.FontSize.footnote, color = Tokens.palette.mutedFg)
        }
        if (onDismiss != null) {
            Icon(
                Icons.Outlined.Close,
                contentDescription = stringResource(R.string.common_dismiss),
                tint = Tokens.palette.mutedFg,
                modifier = Modifier.size(24.dp).clickable(onClick = onDismiss),
            )
        }
    }
}
