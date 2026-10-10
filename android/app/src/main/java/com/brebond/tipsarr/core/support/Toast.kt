package com.brebond.tipsarr.core.support

import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.slideInVertically
import androidx.compose.animation.slideOutVertically
import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.safeDrawingPadding
import androidx.compose.foundation.layout.size
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.Warning
import androidx.compose.material3.Icon
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.compositionLocalOf
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.semantics.LiveRegionMode
import androidx.compose.ui.semantics.liveRegion
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.brebond.tipsarr.design.Tokens
import com.brebond.tipsarr.design.palette
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

enum class ToastKind { Success, Error }

data class ToastMessage(val text: String, val kind: ToastKind)

/** One short message at a time, shown above the content for a few seconds. */
class ToastCenter(private val scope: CoroutineScope) {
    var message by mutableStateOf<ToastMessage?>(null)
        private set
    private var dismiss: Job? = null

    fun show(text: String, kind: ToastKind = ToastKind.Success) {
        val shown = ToastMessage(text, kind)
        message = shown
        dismiss?.cancel()
        dismiss = scope.launch {
            delay(3_000)
            if (message === shown) message = null
        }
    }
}

val LocalToast = compositionLocalOf<ToastCenter?> { null }

@Composable
fun ToastHost(center: ToastCenter, modifier: Modifier = Modifier) {
    val current = center.message
    Box(modifier.fillMaxWidth().safeDrawingPadding().padding(top = Tokens.Spacing.sm), contentAlignment = Alignment.TopCenter) {
        AnimatedVisibility(
            visible = current != null,
            enter = slideInVertically { -it } + fadeIn(),
            exit = slideOutVertically { -it } + fadeOut(),
        ) {
            val shown = current
            if (shown != null) {
                Surface(
                    shape = CircleShape,
                    color = Tokens.palette.card,
                    shadowElevation = 6.dp,
                    border = BorderStroke(1.dp, Tokens.palette.border),
                    modifier = Modifier.padding(horizontal = Tokens.Spacing.lg).semantics { liveRegion = LiveRegionMode.Polite },
                ) {
                    Row(
                        Modifier.padding(horizontal = Tokens.Spacing.lg, vertical = Tokens.Spacing.md),
                        horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm),
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Icon(
                            if (shown.kind == ToastKind.Success) Icons.Filled.CheckCircle else Icons.Filled.Warning,
                            contentDescription = null,
                            tint = if (shown.kind == ToastKind.Success) Tokens.Status.available else Tokens.palette.destructive,
                            modifier = Modifier.size(20.dp),
                        )
                        Text(shown.text, fontSize = Tokens.FontSize.subhead, fontWeight = FontWeight.Medium, color = Tokens.palette.fg)
                    }
                }
            }
        }
    }
}
