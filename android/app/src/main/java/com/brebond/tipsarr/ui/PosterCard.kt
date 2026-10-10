package com.brebond.tipsarr.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Movie
import androidx.compose.material.icons.filled.Visibility
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.semantics.clearAndSetSemantics
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import com.brebond.tipsarr.design.Tokens
import com.brebond.tipsarr.design.palette

/**
 * Poster with title, year, optional score and request state. The image comes from the `poster` slot
 * (the image loader with the account's token arrives with Discover); it gets a film placeholder until then.
 */
@Composable
fun PosterCard(
    title: String,
    modifier: Modifier = Modifier,
    subtitle: String? = null,
    state: RequestState? = null,
    watched: Boolean = false,
    rating: (@Composable () -> Unit)? = null,
    poster: @Composable () -> Unit = {},
) {
    val p = Tokens.palette
    val description = buildString {
        append(title)
        if (subtitle != null) append(", ").append(subtitle)
    }
    val stateLabel = state?.let { stringResource(it.title) }
    Column(
        modifier.fillMaxWidth().clearAndSetSemantics { contentDescription = description + (stateLabel?.let { ", $it" } ?: "") },
        verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm),
    ) {
        Box(
            Modifier.fillMaxWidth().aspectRatio(1f / Tokens.Size.posterRatio).clip(RoundedCornerShape(Tokens.Radius.md)).background(p.muted),
        ) {
            Icon(Icons.Filled.Movie, contentDescription = null, tint = p.mutedFg, modifier = Modifier.align(Alignment.Center).size(28.dp))
            poster()
            if (state != null) StatusBadge(state, Modifier.align(Alignment.TopEnd).padding(Tokens.Spacing.sm), compact = true)
            if (watched) {
                Icon(
                    Icons.Filled.Visibility, contentDescription = null, tint = Color.White,
                    modifier = Modifier.align(Alignment.TopStart).padding(Tokens.Spacing.sm).clip(CircleShape)
                        .background(Color.Black.copy(alpha = 0.55f)).padding(5.dp).size(12.dp),
                )
            }
        }
        Text(title, fontSize = Tokens.FontSize.subhead, fontWeight = FontWeight.Medium, color = p.fg, maxLines = 1, overflow = TextOverflow.Ellipsis)
        if (subtitle != null || rating != null) {
            Row(horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.xs), verticalAlignment = Alignment.CenterVertically) {
                if (subtitle != null) Text(subtitle, fontSize = Tokens.FontSize.footnote, color = p.mutedFg, maxLines = 1, modifier = Modifier.weight(1f))
                if (rating != null) rating()
            }
        }
    }
}
