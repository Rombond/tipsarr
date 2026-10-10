package com.brebond.tipsarr.features.discover

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.GridItemSpan
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.itemsIndexed
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.itemsIndexed
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.AutoAwesome
import androidx.compose.material.icons.outlined.HourglassTop
import androidx.compose.material.icons.outlined.Search
import androidx.compose.material.icons.outlined.WarningAmber
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Text
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.heading
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import com.brebond.tipsarr.R
import com.brebond.tipsarr.core.api.ApiError
import com.brebond.tipsarr.core.api.BoxOfficeChart
import com.brebond.tipsarr.core.api.MediaItem
import com.brebond.tipsarr.core.live.LocalLive
import com.brebond.tipsarr.core.live.OnTick
import com.brebond.tipsarr.design.Tokens
import com.brebond.tipsarr.design.palette
import com.brebond.tipsarr.ui.Chip
import com.brebond.tipsarr.ui.MediaPoster
import com.brebond.tipsarr.ui.OfflineState
import com.brebond.tipsarr.ui.PosterSkeleton
import com.brebond.tipsarr.ui.SkeletonBlock
import com.brebond.tipsarr.ui.StateView
import kotlinx.coroutines.launch

private val GUTTER = Tokens.Spacing.lg

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun DiscoverScreen(model: DiscoverModel, boxOffice: BoxOfficeModel, onOpenChart: (BoxOfficeChart) -> Unit, onOpenItem: (MediaItem) -> Unit, modifier: Modifier = Modifier) {
    var refreshing by remember { mutableStateOf(false) }
    val scope = rememberCoroutineScope()
    val source = model.source(model.chip)
    val live = LocalLive.current
    OnTick(live?.suggestionsTick ?: 0) { if (model.chip == DiscoverChip.ForYou) model.home.refresh() }
    OnTick(live?.requestsTick ?: 0) { model.refreshCurrent() }
    PullToRefreshBox(
        isRefreshing = refreshing,
        onRefresh = { scope.launch { refreshing = true; model.refreshCurrent(); if (model.chip == DiscoverChip.ForYou) boxOffice.refresh(); refreshing = false } },
        modifier = modifier.fillMaxSize(),
    ) {
        if (source == null) ForYou(model, boxOffice, onOpenChart, onOpenItem) else Grid(model, model.list(source), onOpenItem)
    }
}

@Composable
private fun Header(model: DiscoverModel, gutter: Dp = GUTTER) {
    Column(verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.md)) {
        Row(Modifier.fillMaxWidth().padding(start = gutter, end = gutter, top = Tokens.Spacing.sm), verticalAlignment = Alignment.CenterVertically) {
            Text(
                stringResource(R.string.m_tab_discover), fontSize = Tokens.FontSize.largeTitle, fontWeight = FontWeight.Bold,
                color = Tokens.palette.fg, modifier = Modifier.weight(1f).semantics { heading() },
            )
        }
        LazyRow(horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm), contentPadding = PaddingValues(horizontal = gutter)) {
            items(DiscoverChip.entries) { item ->
                Chip(stringResource(item.title()), selected = model.chip == item, onClick = { model.chip = item })
            }
        }
    }
}

private fun DiscoverChip.title(): Int = when (this) {
    DiscoverChip.ForYou -> R.string.m_discover_chip_for_you
    DiscoverChip.Trending -> R.string.discover_trending
    DiscoverChip.Upcoming -> R.string.m_discover_chip_upcoming
    DiscoverChip.Movies -> R.string.m_discover_chip_movies
    DiscoverChip.Tv -> R.string.m_discover_chip_tv
}

/** "For you" chip: the suggestion rows (the server decides which rows, e.g. trending, because you watched). */
@Composable
private fun ForYou(model: DiscoverModel, boxOffice: BoxOfficeModel, onOpenChart: (BoxOfficeChart) -> Unit, onOpenItem: (MediaItem) -> Unit) {
    val home = model.home
    LaunchedEffect(home) { home.loadIfNeeded() }
    LazyColumn(Modifier.fillMaxSize(), contentPadding = PaddingValues(bottom = Tokens.Spacing.lg), verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.lg)) {
        item { Header(model) }
        when (val phase = home.phase) {
            Phase.Idle, Phase.Loading -> item { RailSkeleton() }
            is Phase.Failed -> item { ErrorBlock(phase.error) { home.refresh() } }
            Phase.Loaded -> if (home.rows.isEmpty()) {
                item { BoxOfficeRail(boxOffice, onOpenItem, onOpenChart) }
                item {
                    StateView(
                        if (home.generating) Icons.Outlined.HourglassTop else Icons.Outlined.AutoAwesome,
                        stringResource(if (home.generating) R.string.suggest_preparing else R.string.common_no_results),
                        Modifier.height(360.dp),
                    )
                }
            } else {
                // The box office sits right under the first suggestion row ("Recommended for you").
                itemsIndexed(home.rows, key = { _, row -> row.id }) { index, row ->
                    Column(verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.lg)) {
                        Rail(row.title, row.items, onOpenItem)
                        if (index == 0) BoxOfficeRail(boxOffice, onOpenItem, onOpenChart)
                    }
                }
            }
        }
    }
}

@Composable
private fun Rail(title: String, items: List<MediaItem>, onOpenItem: (MediaItem) -> Unit) {
    Column(verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.md)) {
        Text(
            title, fontSize = Tokens.FontSize.title3, fontWeight = FontWeight.SemiBold, color = Tokens.palette.fg,
            modifier = Modifier.padding(horizontal = GUTTER).semantics { heading() },
        )
        LazyRow(horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.md), contentPadding = PaddingValues(horizontal = GUTTER)) {
            items(items, key = { it.id }) { item -> MediaPoster(item, { onOpenItem(item) }, Modifier.width(120.dp)) }
        }
    }
}

@Composable
private fun RailSkeleton() {
    Column(verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.x2xl)) {
        repeat(2) {
            Column(Modifier.padding(horizontal = GUTTER), verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.md)) {
                SkeletonBlock(Modifier.width(180.dp), height = 22.dp)
                Row(horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.md)) { repeat(3) { PosterSkeleton(Modifier.width(120.dp)) } }
            }
        }
    }
}

/** Grid chips (Trending, Upcoming, Movies, TV): adaptive grid with infinite scroll. */
@Composable
private fun Grid(model: DiscoverModel, list: MediaListModel, onOpenItem: (MediaItem) -> Unit) {
    LaunchedEffect(list) { list.loadIfNeeded() }
    LazyVerticalGrid(
        columns = GridCells.Adaptive(104.dp),
        modifier = Modifier.fillMaxSize(),
        contentPadding = PaddingValues(start = GUTTER, end = GUTTER, bottom = Tokens.Spacing.lg),
        verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.lg),
        horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.md),
    ) {
        item(span = { GridItemSpan(maxLineSpan) }) { Header(model, gutter = 0.dp) }
        when (val phase = list.phase) {
            Phase.Idle, Phase.Loading -> items(9) { PosterSkeleton() }
            is Phase.Failed -> item(span = { GridItemSpan(maxLineSpan) }) { ErrorBlock(phase.error) { list.refresh() } }
            Phase.Loaded -> {
                itemsIndexed(list.items, key = { _, item -> item.id }) { index, item ->
                    LaunchedEffect(index, list.items.size) { list.loadMore(index) }
                    MediaPoster(item, { onOpenItem(item) })
                }
                if (list.loadingMore) item(span = { GridItemSpan(maxLineSpan) }) {
                    Box(Modifier.fillMaxWidth().padding(Tokens.Spacing.lg), contentAlignment = Alignment.Center) { CircularProgressIndicator() }
                }
            }
        }
    }
}

/** Error view for a failed list: offline gets the offline wording, others the server message. */
@Composable
private fun ErrorBlock(error: ApiError, retry: suspend () -> Unit) {
    val scope = rememberCoroutineScope()
    val context = LocalContext.current
    Box(Modifier.fillMaxWidth().height(360.dp)) {
        if (error == ApiError.Unreachable) {
            OfflineState({ scope.launch { retry() } })
        } else {
            StateView(
                Icons.Outlined.WarningAmber, error.message(context),
                actionTitle = stringResource(R.string.common_retry), onAction = { scope.launch { retry() } },
            )
        }
    }
}
