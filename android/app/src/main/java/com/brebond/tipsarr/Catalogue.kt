package com.brebond.tipsarr

import androidx.compose.foundation.background
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.safeDrawingPadding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.tooling.preview.Preview
import androidx.compose.ui.unit.dp
import com.brebond.tipsarr.design.TipsarrTheme
import com.brebond.tipsarr.design.Tokens
import com.brebond.tipsarr.design.palette
import com.brebond.tipsarr.ui.Banner
import com.brebond.tipsarr.ui.BannerKind
import com.brebond.tipsarr.ui.ButtonKind
import com.brebond.tipsarr.ui.Chip
import com.brebond.tipsarr.ui.Field
import com.brebond.tipsarr.ui.NoResultsState
import com.brebond.tipsarr.ui.OfflineState
import com.brebond.tipsarr.ui.PosterCard
import com.brebond.tipsarr.ui.PosterSkeleton
import com.brebond.tipsarr.ui.ProgressBar
import com.brebond.tipsarr.ui.RequestState
import com.brebond.tipsarr.ui.StatusBadge
import com.brebond.tipsarr.ui.TipsarrButton
import com.brebond.tipsarr.ui.TipsarrSheet

/** Every shared component on one screen (the iOS `Catalogue`). Replaced by the launch flow once it exists. */
@Composable
fun Catalogue() {
    var address by remember { mutableStateOf("") }
    var password by remember { mutableStateOf("secret") }
    var chip by remember { mutableStateOf(0) }
    var sheet by remember { mutableStateOf(false) }
    Column(
        Modifier
            .fillMaxSize()
            .background(MaterialTheme.colorScheme.background)
            .safeDrawingPadding()
            .verticalScroll(rememberScrollState())
            .padding(Tokens.Spacing.lg),
        verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.xl),
    ) {
        Text("Tipsarr", style = MaterialTheme.typography.displaySmall, color = Tokens.palette.fg)

        Section("Status badges") {
            Column(verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm)) {
                RequestState.entries.forEach { StatusBadge(it) }
                Row(horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm)) { RequestState.entries.forEach { StatusBadge(it, compact = true) } }
            }
        }
        Section("Buttons") {
            Column(verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.md)) {
                TipsarrButton(stringResource(R.string.media_request), {}, fullWidth = true)
                TipsarrButton(stringResource(R.string.common_cancel), {}, kind = ButtonKind.Secondary, fullWidth = true)
                TipsarrButton(stringResource(R.string.common_retry), {}, kind = ButtonKind.Ghost)
                TipsarrButton(stringResource(R.string.common_delete), {}, kind = ButtonKind.Destructive, fullWidth = true)
                TipsarrButton(stringResource(R.string.common_save), {}, fullWidth = true, enabled = false)
                TipsarrButton(stringResource(R.string.media_request), {}, compact = true)
            }
        }
        Section("Chips") {
            Row(Modifier.horizontalScroll(rememberScrollState()), horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm)) {
                listOf(R.string.m_tab_discover, R.string.m_tab_requests, R.string.m_tab_library).forEachIndexed { i, id ->
                    Chip(stringResource(id), selected = chip == i, onClick = { chip = i }, count = if (i == 1) 27 else null)
                }
            }
        }
        Section("Progress") { ProgressBar(45); ProgressBar(100, tint = Tokens.Status.available) }
        Section("Banners") {
            Column(verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.md)) {
                Banner(stringResource(R.string.banner_dry_run), message = stringResource(R.string.banner_dry_run_long))
                Banner(stringResource(R.string.m_update_title), kind = BannerKind.Warning, message = stringResource(R.string.m_update_body))
                Banner(stringResource(R.string.m_offline_title), kind = BannerKind.Error, message = stringResource(R.string.m_offline_body), onDismiss = {})
            }
        }
        Section("Fields") {
            Column(verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.lg)) {
                Field(stringResource(R.string.m_connect_address), address, { address = it }, placeholder = stringResource(R.string.m_connect_placeholder))
                Field(stringResource(R.string.m_connect_address), password, { password = it }, secure = true, error = stringResource(R.string.m_connect_unreachable))
            }
        }
        Section("Poster cards and skeletons") {
            Row(horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.md)) {
                PosterCard("Blade Runner 2049", Modifier.weight(1f), subtitle = "2017", state = RequestState.Available)
                PosterCard("Severance", Modifier.weight(1f), subtitle = "2022", state = RequestState.Requested, watched = true)
                PosterCard("A very long title that gets truncated", Modifier.weight(1f), subtitle = "1999")
            }
            Row(horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.md)) { repeat(3) { PosterSkeleton(Modifier.weight(1f)) } }
        }
        Section("States") {
            Column(Modifier.width(320.dp)) {
                NoResultsState(Modifier.padding(0.dp))
            }
        }
        Section("Sheet") { TipsarrButton("Open sheet", { sheet = true }, kind = ButtonKind.Secondary) }
    }
    if (sheet) {
        TipsarrSheet(onDismiss = { sheet = false }) {
            OfflineState(onRetry = { sheet = false }, modifier = Modifier.padding(bottom = Tokens.Spacing.xl))
        }
    }
}

@Composable
private fun Section(title: String, content: @Composable () -> Unit) {
    Column(verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.md)) {
        Text(title, fontSize = Tokens.FontSize.title3, fontWeight = FontWeight.Bold, color = Tokens.palette.fg)
        content()
    }
}

@Preview(showBackground = true)
@Composable
private fun CataloguePreview() = TipsarrTheme(dark = false) { Catalogue() }

@Preview(showBackground = true, uiMode = android.content.res.Configuration.UI_MODE_NIGHT_YES)
@Composable
private fun CatalogueDarkPreview() = TipsarrTheme(dark = true) { Catalogue() }
