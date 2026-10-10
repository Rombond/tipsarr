package com.brebond.tipsarr.features.library

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.defaultMinSize
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.GridItemSpan
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.itemsIndexed
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.FilterAlt
import androidx.compose.material.icons.filled.Visibility
import androidx.compose.material.icons.outlined.Close
import androidx.compose.material.icons.outlined.FilterAlt
import androidx.compose.material.icons.outlined.GridView
import androidx.compose.material.icons.outlined.Movie
import androidx.compose.material.icons.outlined.Search
import androidx.compose.material.icons.outlined.VideoLibrary
import androidx.compose.material.icons.outlined.ViewList
import androidx.compose.material.icons.outlined.WarningAmber
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.OutlinedTextFieldDefaults
import androidx.compose.material3.Text
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.LocalFocusManager
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.heading
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.brebond.tipsarr.R
import com.brebond.tipsarr.core.api.ApiError
import com.brebond.tipsarr.core.api.LibraryItem
import com.brebond.tipsarr.core.api.LibraryKind
import com.brebond.tipsarr.core.api.MediaType
import com.brebond.tipsarr.core.api.WatchedFilter
import com.brebond.tipsarr.core.images.RemoteImage
import com.brebond.tipsarr.core.images.TmdbSize
import com.brebond.tipsarr.core.support.plural
import com.brebond.tipsarr.design.Tokens
import com.brebond.tipsarr.design.palette
import com.brebond.tipsarr.features.detail.MediaRoute
import com.brebond.tipsarr.features.discover.Phase
import com.brebond.tipsarr.ui.Chip
import com.brebond.tipsarr.ui.NoResultsState
import com.brebond.tipsarr.ui.OfflineState
import com.brebond.tipsarr.ui.PosterCard
import com.brebond.tipsarr.ui.PosterSkeleton
import com.brebond.tipsarr.ui.ScoreLabel
import com.brebond.tipsarr.ui.StateView
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

private val GUTTER = Tokens.Spacing.lg

/** Library tab: what is in Jellyfin, with kind chips, search, filters and a grid or list layout. */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun LibraryScreen(model: LibraryModel, onOpen: (MediaRoute) -> Unit, modifier: Modifier = Modifier) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    var showFilters by remember { mutableStateOf(false) }
    var listLayout by rememberSaveable { mutableStateOf(false) }
    var refreshing by remember { mutableStateOf(false) }
    val filters = model.filters

    // Typing in the search field changes `filters`: wait a moment so only the last text is sent.
    LaunchedEffect(filters) {
        if (filters.query.isNotEmpty()) delay(350)
        model.reload()
    }
    LaunchedEffect(Unit) { model.loadFacets() }

    PullToRefreshBox(
        isRefreshing = refreshing,
        onRefresh = { scope.launch { refreshing = true; model.reload(); refreshing = false } },
        modifier = modifier.fillMaxSize(),
    ) {
        LazyVerticalGrid(
            columns = if (listLayout) GridCells.Fixed(1) else GridCells.Adaptive(104.dp),
            modifier = Modifier.fillMaxSize(),
            contentPadding = PaddingValues(start = GUTTER, end = GUTTER, bottom = Tokens.Spacing.lg),
            verticalArrangement = Arrangement.spacedBy(if (listLayout) 0.dp else Tokens.Spacing.lg),
            horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.md),
        ) {
            item(span = { GridItemSpan(maxLineSpan) }) {
                Header(model, listLayout, onToggleLayout = { listLayout = !listLayout }, onFilters = { showFilters = true })
            }
            when (val phase = model.phase) {
                Phase.Idle, Phase.Loading -> items(9) { PosterSkeleton() }
                is Phase.Failed -> item(span = { GridItemSpan(maxLineSpan) }) {
                    Box(Modifier.fillMaxWidth().height(360.dp)) {
                        if (phase.error == ApiError.Unreachable) OfflineState({ scope.launch { model.reload() } })
                        else StateView(Icons.Outlined.WarningAmber, phase.error.message(context), actionTitle = stringResource(R.string.common_retry), onAction = { scope.launch { model.reload() } })
                    }
                }
                Phase.Loaded -> when {
                    model.neverSynced -> item(span = { GridItemSpan(maxLineSpan) }) {
                        StateView(Icons.Outlined.VideoLibrary, stringResource(R.string.library_title), Modifier.height(360.dp), message = stringResource(R.string.library_empty))
                    }
                    model.items.isEmpty() -> item(span = { GridItemSpan(maxLineSpan) }) { NoResultsState(Modifier.height(360.dp)) }
                    else -> {
                        itemsIndexed(model.items, key = { _, item -> item.id }) { index, item ->
                            LaunchedEffect(index, model.items.size) { model.loadMore(index) }
                            val open = { onOpen(MediaRoute(item.type, item.tmdbId, item.title)) }
                            if (listLayout) {
                                Column { LibraryRow(item, open); HorizontalDivider(color = Tokens.palette.border) }
                            } else {
                                PosterCard(
                                    title = item.title, modifier = Modifier.clickable(onClick = open), subtitle = item.year?.toString(), watched = item.watched,
                                    rating = item.rating?.takeIf { it > 0 }?.let { score -> { ScoreLabel(score) } },
                                    poster = { RemoteImage(item.posterPath, TmdbSize.W342, Modifier.fillMaxSize(), server = true) },
                                )
                            }
                        }
                        if (model.loadingMore) item(span = { GridItemSpan(maxLineSpan) }) {
                            Box(Modifier.fillMaxWidth().padding(Tokens.Spacing.lg), contentAlignment = Alignment.Center) { CircularProgressIndicator() }
                        }
                    }
                }
            }
        }
    }

    if (showFilters) LibraryFiltersSheet(model.filters, model.facets, onDismiss = { showFilters = false }) { model.filters = it }
}

@Composable
private fun Header(model: LibraryModel, listLayout: Boolean, onToggleLayout: () -> Unit, onFilters: () -> Unit) {
    val filters = model.filters
    val focus = LocalFocusManager.current
    Column(verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.md), modifier = Modifier.padding(top = Tokens.Spacing.sm)) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Text(
                stringResource(R.string.m_tab_library), fontSize = Tokens.FontSize.largeTitle, fontWeight = FontWeight.Bold,
                color = Tokens.palette.fg, modifier = Modifier.weight(1f).semantics { heading() },
            )
            IconButton(onToggleLayout) {
                Icon(
                    if (listLayout) Icons.Outlined.GridView else Icons.Outlined.ViewList,
                    stringResource(if (listLayout) R.string.m_library_grid else R.string.m_library_list), tint = Tokens.palette.fg,
                )
            }
            IconButton(onFilters) {
                Icon(
                    if (filters.sheetIsDefault) Icons.Outlined.FilterAlt else Icons.Filled.FilterAlt,
                    stringResource(R.string.library_filters), tint = Tokens.palette.fg,
                )
            }
        }
        OutlinedTextField(
            value = filters.query,
            onValueChange = { model.filters = filters.copy(query = it) },
            modifier = Modifier.fillMaxWidth().defaultMinSize(minHeight = Tokens.Size.touchTarget),
            placeholder = { Text(stringResource(R.string.library_search), color = Tokens.palette.mutedFg) },
            leadingIcon = { Icon(Icons.Outlined.Search, null, tint = Tokens.palette.mutedFg) },
            trailingIcon = {
                if (filters.query.isNotEmpty()) IconButton({ model.filters = filters.copy(query = "") }) { Icon(Icons.Outlined.Close, stringResource(R.string.common_dismiss), tint = Tokens.palette.mutedFg) }
            },
            singleLine = true,
            keyboardOptions = KeyboardOptions(imeAction = ImeAction.Search),
            keyboardActions = KeyboardActions(onSearch = { focus.clearFocus() }),
            shape = CircleShape,
            colors = OutlinedTextFieldDefaults.colors(
                focusedContainerColor = Tokens.palette.muted, unfocusedContainerColor = Tokens.palette.muted,
                focusedBorderColor = Tokens.palette.ring, unfocusedBorderColor = Tokens.palette.muted,
                focusedTextColor = Tokens.palette.fg, unfocusedTextColor = Tokens.palette.fg, cursorColor = Tokens.palette.fg,
            ),
        )
        LazyRow(horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm)) {
            item {
                Chip(stringResource(R.string.library_all), selected = filters.kind == LibraryKind.All && filters.watched != WatchedFilter.No, onClick = {
                    model.filters = filters.copy(kind = LibraryKind.All, watched = WatchedFilter.Any)
                })
            }
            item { Chip(stringResource(R.string.type_movies), selected = filters.kind == LibraryKind.Movie, onClick = { model.filters = filters.copy(kind = LibraryKind.Movie) }) }
            item { Chip(stringResource(R.string.type_shows), selected = filters.kind == LibraryKind.Tv, onClick = { model.filters = filters.copy(kind = LibraryKind.Tv) }) }
            item {
                Chip(stringResource(R.string.m_library_unwatched), selected = filters.watched == WatchedFilter.No, onClick = {
                    model.filters = filters.copy(watched = if (filters.watched == WatchedFilter.No) WatchedFilter.Any else WatchedFilter.No)
                })
            }
        }
    }
}

@Composable
private fun LibraryRow(item: LibraryItem, onClick: () -> Unit) {
    val context = LocalContext.current
    val meta = listOfNotNull(
        item.year?.toString(),
        stringResource(if (item.type == MediaType.Tv) R.string.type_tv else R.string.type_movie),
        item.runtimeMinutes?.takeIf { it > 0 }?.let { context.plural("time_min", it) },
    ).joinToString(" · ")
    Row(
        Modifier.fillMaxWidth().clickable(onClick = onClick).padding(vertical = Tokens.Spacing.sm),
        horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.md), verticalAlignment = Alignment.CenterVertically,
    ) {
        Box(Modifier.width(56.dp).height(84.dp).clip(RoundedCornerShape(Tokens.Radius.sm)).background(Tokens.palette.muted), contentAlignment = Alignment.Center) {
            Icon(Icons.Outlined.Movie, null, tint = Tokens.palette.mutedFg)
            RemoteImage(item.posterPath, TmdbSize.W185, Modifier.fillMaxSize(), server = true)
        }
        Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.xs)) {
            Text(item.title, fontSize = Tokens.FontSize.body, fontWeight = FontWeight.SemiBold, color = Tokens.palette.fg, maxLines = 2, overflow = TextOverflow.Ellipsis)
            Text(meta, fontSize = Tokens.FontSize.footnote, color = Tokens.palette.mutedFg, maxLines = 1)
            if (item.watched) {
                Row(horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.xs), verticalAlignment = Alignment.CenterVertically) {
                    Icon(Icons.Filled.Visibility, null, tint = Tokens.palette.mutedFg, modifier = Modifier.size(14.dp))
                    Text(stringResource(R.string.library_watched_state), fontSize = Tokens.FontSize.caption, color = Tokens.palette.mutedFg)
                }
            }
        }
        item.rating?.takeIf { it > 0 }?.let { ScoreLabel(it) }
    }
}
