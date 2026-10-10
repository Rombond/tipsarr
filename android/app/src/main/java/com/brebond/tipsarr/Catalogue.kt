package com.brebond.tipsarr

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.safeDrawingPadding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import com.brebond.tipsarr.design.Tokens
import com.brebond.tipsarr.design.palette

/** First screen of the Android app: the design tokens and a string from the shared catalogue, until the launch flow exists. */
@Composable
fun Catalogue() {
    Column(
        Modifier
            .fillMaxSize()
            .background(MaterialTheme.colorScheme.background)
            .safeDrawingPadding()
            .verticalScroll(rememberScrollState())
            .padding(Tokens.Spacing.lg),
        verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.lg),
    ) {
        Text("Tipsarr", style = MaterialTheme.typography.displaySmall, color = MaterialTheme.colorScheme.onBackground)
        Text(stringResource(R.string.req_stage_downloading_pct, "45"), color = Tokens.palette.mutedFg)
        Text("Status", color = Tokens.palette.mutedFg)
        Row(horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm)) {
            listOf(
                Tokens.Status.available, Tokens.Status.partial, Tokens.Status.requested, Tokens.Status.approved,
                Tokens.Status.searching, Tokens.Status.downloading, Tokens.Status.declined, Tokens.Status.failed,
            ).forEach { Box(Modifier.size(32.dp).clip(RoundedCornerShape(Tokens.Radius.sm)).background(it)) }
        }
    }
}
