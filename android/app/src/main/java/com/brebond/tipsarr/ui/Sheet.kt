package com.brebond.tipsarr.ui

import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.material3.BottomSheetDefaults
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.foundation.shape.RoundedCornerShape
import com.brebond.tipsarr.design.Tokens
import com.brebond.tipsarr.design.palette

/** Shared sheet look: token corner radius, grabber, app background. `fullHeight` skips the half-height stop. */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun TipsarrSheet(onDismiss: () -> Unit, fullHeight: Boolean = false, content: @Composable ColumnScope.() -> Unit) {
    ModalBottomSheet(
        onDismissRequest = onDismiss,
        sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = fullHeight),
        shape = RoundedCornerShape(topStart = Tokens.Radius.sheet, topEnd = Tokens.Radius.sheet),
        containerColor = Tokens.palette.bg,
        dragHandle = { BottomSheetDefaults.DragHandle() },
        content = content,
    )
}
