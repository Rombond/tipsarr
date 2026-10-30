package com.brebond.tipsarr.features.search

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.GridItemSpan
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.itemsIndexed
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.WarningAmber
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import com.brebond.tipsarr.R
import com.brebond.tipsarr.core.api.ApiError
import com.brebond.tipsarr.core.api.MediaItem
import com.brebond.tipsarr.core.api.MediaType
import com.brebond.tipsarr.design.Tokens
import com.brebond.tipsarr.features.discover.MediaListModel
import com.brebond.tipsarr.features.discover.Phase
import com.brebond.tipsarr.ui.Chip
import com.brebond.tipsarr.ui.MediaPoster
import com.brebond.tipsarr.ui.OfflineState
import com.brebond.tipsarr.ui.PosterSkeleton
import com.brebond.tipsarr.ui.StateView
import com.brebond.tipsarr.ui.TopBarScaffold
import kotlinx.coroutines.launch

/** One genre by name: the TMDB id for movies and/or the one for shows. */
data class GenreRoute(val name: String, val movieId: Int?, val tvId: Int?)

/** Titles of one genre, with infinite scroll; a Movies / Shows switch when the genre exists for both. */
@Composable
fun GenreScreen(route: GenreRoute, movies: MediaListModel?, shows: MediaListModel?, onBack: () -> Unit, onOpenItem: (MediaItem) -> Unit) {
    var type by rememberSaveable { mutableStateOf(if (movies != null) MediaType.Movie else MediaType.Tv) }
    val list = (if (type == MediaType.Movie) movies else shows) ?: return
    val scope = rememberCoroutineScope()
    val context = LocalContext.current
    LaunchedEffect(list) { list.loadIfNeeded() }
    TopBarScaffold(route.name, onBack) {
        if (movies != null && shows != null) {
            Row(Modifier.padding(horizontal = Tokens.Spacing.lg), horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm)) {
                Chip(stringResource(R.string.type_movies), selected = type == MediaType.Movie, onClick = { type = MediaType.Movie })
                Chip(stringResource(R.string.type_shows), selected = type == MediaType.Tv, onClick = { type = MediaType.Tv })
            }
        }
        when (val phase = list.phase) {
            Phase.Idle, Phase.Loading -> LazyVerticalGrid(
                GridCells.Adaptive(96.dp), Modifier.fillMaxSize(), contentPadding = PaddingValues(Tokens.Spacing.lg),
                verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.lg), horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.md),
            ) { items(9) { PosterSkeleton() } }
            is Phase.Failed -> Box(Modifier.fillMaxSize()) {
                if (phase.error == ApiError.Unreachable) OfflineState({ scope.launch { list.refresh() } })
                else StateView(Icons.Outlined.WarningAmber, phase.error.message(context), actionTitle = stringResource(R.string.common_retry), onAction = { scope.launch { list.refresh() } })
            }
            Phase.Loaded -> LazyVerticalGrid(
                GridCells.Adaptive(96.dp), Modifier.fillMaxSize(), contentPadding = PaddingValues(Tokens.Spacing.lg),
                verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.lg), horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.md),
            ) {
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
