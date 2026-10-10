package com.brebond.tipsarr.ui

import android.provider.Settings
import androidx.compose.animation.core.RepeatMode
import androidx.compose.animation.core.animateFloat
import androidx.compose.animation.core.infiniteRepeatable
import androidx.compose.animation.core.rememberInfiniteTransition
import androidx.compose.animation.core.tween
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.clearAndSetSemantics
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import com.brebond.tipsarr.R
import com.brebond.tipsarr.design.Tokens
import com.brebond.tipsarr.design.palette

/** True when the user turned animations off (system "Remove animations"): shimmer stays still. */
@Composable
private fun reduceMotion(): Boolean {
    val context = LocalContext.current
    return remember { Settings.Global.getFloat(context.contentResolver, Settings.Global.ANIMATOR_DURATION_SCALE, 1f) == 0f }
}

/** Placeholder shimmer around loading content; announced as "Loading…". */
@Composable
fun Skeleton(modifier: Modifier = Modifier, content: @Composable () -> Unit) {
    val label = stringResource(R.string.common_loading)
    val alpha = if (reduceMotion()) {
        0.6f
    } else {
        val a by rememberInfiniteTransition(label = "skeleton").animateFloat(
            initialValue = 0.8f, targetValue = 0.4f,
            animationSpec = infiniteRepeatable(tween(900), RepeatMode.Reverse), label = "alpha",
        )
        a
    }
    Box(modifier.alpha(alpha).clearAndSetSemantics { contentDescription = label }) { content() }
}

@Composable
fun SkeletonBlock(modifier: Modifier = Modifier, height: Dp = 16.dp, cornerRadius: Dp = Tokens.Radius.sm) {
    Box(modifier.fillMaxWidth().height(height).clip(RoundedCornerShape(cornerRadius)).background(Tokens.palette.border))
}

@Composable
fun PosterSkeleton(modifier: Modifier = Modifier) {
    Skeleton(modifier) {
        Column(verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm)) {
            Box(
                Modifier.fillMaxWidth().aspectRatio(1f / Tokens.Size.posterRatio)
                    .clip(RoundedCornerShape(Tokens.Radius.md)).background(Tokens.palette.border),
            )
            SkeletonBlock(height = 14.dp)
            SkeletonBlock(Modifier.width(60.dp), height = 12.dp)
        }
    }
}
