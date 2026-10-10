package com.brebond.tipsarr.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.clearAndSetSemantics
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.brebond.tipsarr.design.Tokens

/** Pill with the state's symbol and name; `compact` keeps the symbol only (poster corner). Colour is never the only signal. */
@Composable
fun StatusBadge(state: RequestState, modifier: Modifier = Modifier, compact: Boolean = false) {
    val label = stringResource(state.title)
    Row(
        modifier
            .clip(CircleShape)
            .background(state.color.copy(alpha = 0.15f))
            .padding(horizontal = if (compact) Tokens.Spacing.xs else Tokens.Spacing.sm, vertical = Tokens.Spacing.xs)
            .clearAndSetSemantics { contentDescription = label },
        horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.xs),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Icon(state.icon, contentDescription = null, tint = state.color, modifier = Modifier.size(14.dp))
        if (!compact) Text(label, color = state.color, fontSize = Tokens.FontSize.caption, fontWeight = FontWeight.SemiBold)
    }
}
