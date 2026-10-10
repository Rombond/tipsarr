package com.brebond.tipsarr.design

import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.staticCompositionLocalOf

/** The neutral palette of the current theme, for colours Material 3 has no slot for (muted, border...). */
val LocalPalette = staticCompositionLocalOf { Tokens.light }

/** Shortcut: `Tokens.palette.mutedFg` inside a composable. */
val Tokens.palette: Tokens.Palette
    @Composable get() = LocalPalette.current

@Composable
fun TipsarrTheme(dark: Boolean = isSystemInDarkTheme(), content: @Composable () -> Unit) {
    val p = if (dark) Tokens.dark else Tokens.light
    val scheme = if (dark) {
        darkColorScheme(
            background = p.bg, onBackground = p.fg, surface = p.bg, onSurface = p.fg,
            surfaceVariant = p.muted, onSurfaceVariant = p.mutedFg, surfaceContainer = p.card,
            primary = p.primary, onPrimary = p.primaryFg, error = p.destructive, outline = p.border,
        )
    } else {
        lightColorScheme(
            background = p.bg, onBackground = p.fg, surface = p.bg, onSurface = p.fg,
            surfaceVariant = p.muted, onSurfaceVariant = p.mutedFg, surfaceContainer = p.card,
            primary = p.primary, onPrimary = p.primaryFg, error = p.destructive, outline = p.border,
        )
    }
    CompositionLocalProvider(LocalPalette provides p) {
        MaterialTheme(colorScheme = scheme, content = content)
    }
}
