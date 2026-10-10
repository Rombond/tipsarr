package com.brebond.tipsarr.features.system

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.safeDrawingPadding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.semantics.heading
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.brebond.tipsarr.design.Tokens
import com.brebond.tipsarr.design.palette

/** Shared layout of the connect / login screens: a centred column that stays readable on tablets and foldables. */
@Composable
fun AuthScaffold(icon: ImageVector, title: String, subtitle: String? = null, content: @Composable () -> Unit) {
    Box(
        Modifier.fillMaxSize().background(Tokens.palette.bg).safeDrawingPadding().imePadding().verticalScroll(rememberScrollState()),
        contentAlignment = Alignment.TopCenter,
    ) {
        Column(
            Modifier.widthIn(max = 480.dp).fillMaxWidth().padding(Tokens.Spacing.x2xl),
            verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.x2xl),
        ) {
            Column(verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.md)) {
                Box(
                    Modifier.size(56.dp).clip(RoundedCornerShape(Tokens.Radius.lg)).background(Tokens.palette.primary),
                    contentAlignment = Alignment.Center,
                ) { Icon(icon, contentDescription = null, tint = Tokens.palette.primaryFg, modifier = Modifier.size(28.dp)) }
                Text(title, fontSize = Tokens.FontSize.largeTitle, fontWeight = FontWeight.Bold, color = Tokens.palette.fg, modifier = Modifier.semantics { heading() })
                if (subtitle != null) Text(subtitle, fontSize = Tokens.FontSize.body, color = Tokens.palette.mutedFg)
            }
            content()
        }
    }
}
