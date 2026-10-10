package com.brebond.tipsarr.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.defaultMinSize
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.semantics.selected
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.brebond.tipsarr.design.Tokens
import com.brebond.tipsarr.design.palette

/** Selectable filter chip (Discover genres, Requests filters); `count` is shown after the title when above zero. */
@Composable
fun Chip(title: String, selected: Boolean, onClick: () -> Unit, modifier: Modifier = Modifier, count: Int? = null) {
    val p = Tokens.palette
    Row(
        modifier
            .defaultMinSize(minHeight = Tokens.Size.touchTarget - Tokens.Spacing.sm)
            .clip(CircleShape)
            .background(if (selected) p.primary else p.muted)
            .clickable(role = Role.Tab, onClick = onClick)
            .semantics { this.selected = selected }
            .padding(horizontal = Tokens.Spacing.md),
        horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.xs),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        val color = if (selected) p.primaryFg else p.fg
        Text(title, color = color, fontSize = Tokens.FontSize.subhead, fontWeight = FontWeight.Medium)
        if (count != null && count > 0) {
            Text(count.toString(), color = color, fontSize = Tokens.FontSize.caption, fontWeight = FontWeight.SemiBold, modifier = Modifier.alpha(0.7f))
        }
    }
}
