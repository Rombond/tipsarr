package com.brebond.tipsarr.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.semantics.ProgressBarRangeInfo
import androidx.compose.ui.semantics.progressBarRangeInfo
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.unit.dp
import com.brebond.tipsarr.design.Tokens
import com.brebond.tipsarr.design.palette

/** Thin download bar, 0...100. */
@Composable
fun ProgressBar(percent: Int, modifier: Modifier = Modifier, tint: Color = Tokens.Status.downloading) {
    val value = percent.coerceIn(0, 100)
    Box(
        modifier
            .fillMaxWidth()
            .height(6.dp)
            .clip(CircleShape)
            .background(Tokens.palette.muted)
            .semantics { progressBarRangeInfo = ProgressBarRangeInfo(value.toFloat(), 0f..100f) },
    ) {
        Box(Modifier.fillMaxHeight().fillMaxWidth(value / 100f).clip(CircleShape).background(tint))
    }
}
