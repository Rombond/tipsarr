package com.brebond.tipsarr.core.support

import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp

/** Width of a screen's content area (the iOS `ContentWidth`): phone column, one wide column (tablet portrait), two columns (landscape). */
enum class ContentWidth {
    Compact, Regular, Wide;

    companion object {
        fun of(width: Dp): ContentWidth = if (width < 620.dp) Compact else if (width < 900.dp) Regular else Wide
    }
}
