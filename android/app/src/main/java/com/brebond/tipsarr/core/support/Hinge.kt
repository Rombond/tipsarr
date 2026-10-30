package com.brebond.tipsarr.core.support

import android.app.Activity
import android.content.Context
import android.content.ContextWrapper
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.layout.onGloballyPositioned
import androidx.compose.ui.layout.positionInWindow
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import androidx.window.layout.FoldingFeature
import androidx.window.layout.WindowInfoTracker

/** A vertical fold or hinge that splits the window: the left pane ends at `left`, the right pane starts at `right` (window dp). */
data class Hinge(val left: Dp, val right: Dp)

/** A horizontal fold, half open (table-top posture): the upper half ends at `top`, the lower half starts at `bottom` (window dp). */
data class TableTop(val top: Dp, val bottom: Dp)

private tailrec fun Context.findActivity(): Activity? = when (this) {
    is Activity -> this
    is ContextWrapper -> baseContext.findActivity()
    else -> null
}

/** Debug aid: `am start ... --es <name> 600,640` (two edges in dp) to try a fold layout on a flat screen. */
private fun Activity.fakeFold(name: String): Pair<Dp, Dp>? {
    if (!com.brebond.tipsarr.BuildConfig.DEBUG) return null
    return intent?.getStringExtra(name)?.split(',')?.mapNotNull { it.trim().toFloatOrNull() }?.takeIf { it.size == 2 }?.let { Dp(it[0]) to Dp(it[1]) }
}

/** The window's fold, if any (null on a flat screen or before the first layout info). */
@Composable
private fun rememberFoldingFeature(activity: Activity): FoldingFeature? {
    val flow = remember(activity) { WindowInfoTracker.getOrCreate(activity).windowLayoutInfo(activity) }
    val info by flow.collectAsState(initial = null)
    return info?.displayFeatures?.filterIsInstance<FoldingFeature>()?.firstOrNull()
}

/** The separating vertical fold of a foldable window (the iOS `FoldInfo.verticalHinge`), or null on a flat screen. */
@Composable
fun rememberVerticalHinge(): Hinge? {
    val activity = LocalContext.current.findActivity() ?: return null
    val density = LocalDensity.current
    activity.fakeFold("fakeHinge")?.let { return Hinge(it.first, it.second) }
    val feature = rememberFoldingFeature(activity)?.takeIf { it.isSeparating && it.orientation == FoldingFeature.Orientation.VERTICAL } ?: return null
    return with(density) { Hinge(feature.bounds.left.toDp(), feature.bounds.right.toDp()) }
}

/**
 * The fold of a foldable half opened with a horizontal hinge (table-top, like a laptop), or null otherwise
 * (the iOS laptop-posture rule: content in the upper half, controls in the lower half).
 */
@Composable
fun rememberTableTop(): TableTop? {
    val activity = LocalContext.current.findActivity() ?: return null
    val density = LocalDensity.current
    activity.fakeFold("fakeTableTop")?.let { return TableTop(it.first, it.second) }
    val feature = rememberFoldingFeature(activity)
        ?.takeIf { it.state == FoldingFeature.State.HALF_OPENED && it.orientation == FoldingFeature.Orientation.HORIZONTAL } ?: return null
    return with(density) { TableTop(feature.bounds.top.toDp(), feature.bounds.bottom.toDp()) }
}

/** The table-top fold in this layout's own coordinates: where it sits in the window, minus where the layout starts. */
@Composable
fun Modifier.foldOffset(onOffset: (Dp) -> Unit): Modifier {
    val density = LocalDensity.current
    return onGloballyPositioned { onOffset(with(density) { it.positionInWindow().y.toDp() }) }
}

/** Splits at a table-top fold: [upper] above it, [lower] below it, nothing on the fold itself. */
@Composable
fun TableTopSplit(fold: TableTop, modifier: Modifier = Modifier, upper: @Composable () -> Unit, lower: @Composable () -> Unit) {
    var offset by remember { mutableStateOf(0.dp) }
    Column(modifier.fillMaxSize().foldOffset { offset = it }) {
        Box(Modifier.fillMaxWidth().height((fold.top - offset).coerceAtLeast(0.dp))) { upper() }
        Spacer(Modifier.height(fold.bottom - fold.top))
        Box(Modifier.fillMaxWidth().weight(1f)) { lower() }
    }
}
