package com.brebond.tipsarr.core.support

import android.app.Activity
import android.content.Context
import android.content.ContextWrapper
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.unit.Dp
import androidx.window.layout.FoldingFeature
import androidx.window.layout.WindowInfoTracker

/** A vertical fold or hinge that splits the window: the left pane ends at `left`, the right pane starts at `right` (window dp). */
data class Hinge(val left: Dp, val right: Dp)

private tailrec fun Context.findActivity(): Activity? = when (this) {
    is Activity -> this
    is ContextWrapper -> baseContext.findActivity()
    else -> null
}

/** The separating vertical fold of a foldable window (the iOS `FoldInfo.verticalHinge`), or null on a flat screen. */
@Composable
fun rememberVerticalHinge(): Hinge? {
    val activity = LocalContext.current.findActivity() ?: return null
    val density = LocalDensity.current
    // Debug aid: `am start ... --es fakeHinge 600,640` (left and right edge in dp) to try the two-pane layout on a flat screen.
    if (com.brebond.tipsarr.BuildConfig.DEBUG) {
        activity.intent?.getStringExtra("fakeHinge")?.split(',')?.mapNotNull { it.trim().toFloatOrNull() }?.takeIf { it.size == 2 }
            ?.let { return Hinge(Dp(it[0]), Dp(it[1])) }
    }
    val flow = remember(activity) { WindowInfoTracker.getOrCreate(activity).windowLayoutInfo(activity) }
    val info by flow.collectAsState(initial = null)
    val feature = info?.displayFeatures?.filterIsInstance<FoldingFeature>()
        ?.firstOrNull { it.isSeparating && it.orientation == FoldingFeature.Orientation.VERTICAL } ?: return null
    return with(density) { Hinge(feature.bounds.left.toDp(), feature.bounds.right.toDp()) }
}
