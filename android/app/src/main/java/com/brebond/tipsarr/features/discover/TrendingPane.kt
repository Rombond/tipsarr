package com.brebond.tipsarr.features.discover

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.GridItemSpan
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.itemsIndexed
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import com.brebond.tipsarr.R
import com.brebond.tipsarr.core.api.MediaItem
import com.brebond.tipsarr.design.Tokens
import com.brebond.tipsarr.ui.MediaPoster
import com.brebond.tipsarr.ui.PosterSkeleton
import com.brebond.tipsarr.ui.TopBarScaffold

/** Search tab, right pane, nothing picked yet: the trending titles. */
@Composable
fun TrendingPane(list: MediaListModel, onOpenItem: (MediaItem) -> Unit) {
    LaunchedEffect(list) { list.loadIfNeeded() }
    TopBarScaffold(stringResource(R.string.m_pane_trending), onBack = {}) {
        LazyVerticalGrid(
            GridCells.Adaptive(104.dp), Modifier.fillMaxSize(), contentPadding = PaddingValues(Tokens.Spacing.lg),
            verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.lg), horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.md),
        ) {
            if (list.phase != Phase.Loaded) items(9) { PosterSkeleton() }
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
