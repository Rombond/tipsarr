package com.brebond.tipsarr.features.settings

import androidx.compose.foundation.ExperimentalFoundationApi
import androidx.compose.foundation.combinedClickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.GridItemSpan
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.Inbox
import androidx.compose.material.icons.outlined.WarningAmber
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Text
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import com.brebond.tipsarr.R
import com.brebond.tipsarr.core.api.ApiError
import com.brebond.tipsarr.core.api.MediaItem
import com.brebond.tipsarr.core.api.TipsarrApi
import com.brebond.tipsarr.core.api.blocklist
import com.brebond.tipsarr.core.api.setBlocklisted
import com.brebond.tipsarr.core.api.setWatchlisted
import com.brebond.tipsarr.core.api.watchlist
import com.brebond.tipsarr.core.support.LocalToast
import com.brebond.tipsarr.core.support.ToastKind
import com.brebond.tipsarr.design.Tokens
import com.brebond.tipsarr.design.palette
import com.brebond.tipsarr.ui.MediaPoster
import com.brebond.tipsarr.ui.OfflineState
import com.brebond.tipsarr.ui.PosterSkeleton
import com.brebond.tipsarr.ui.StateView
import com.brebond.tipsarr.ui.TopBarScaffold
import kotlinx.coroutines.launch

/** Titles saved with "Watchlist". Long press removes one. */
@Composable
fun WatchlistScreen(api: TipsarrApi, onBack: () -> Unit, onOpen: (MediaItem) -> Unit) {
    val context = LocalContext.current
    TitleGridScreen(
        title = stringResource(R.string.watchlist_title), hint = null, emptyMessage = stringResource(R.string.watchlist_empty),
        actionTitle = stringResource(R.string.m_library_remove), doneMessage = context.getString(R.string.actions_toast_watch_removed),
        load = { api.watchlist() }, action = { api.setWatchlisted(false, it.type, it.tmdbId) }, onBack = onBack, onOpen = onOpen,
    )
}

/** Titles marked "not interested". Long press shows one again. */
@Composable
fun HiddenScreen(api: TipsarrApi, onBack: () -> Unit, onOpen: (MediaItem) -> Unit) {
    val context = LocalContext.current
    TitleGridScreen(
        title = stringResource(R.string.profile_hidden_title), hint = stringResource(R.string.profile_hidden_desc), emptyMessage = stringResource(R.string.profile_nothing_hidden),
        actionTitle = stringResource(R.string.m_library_show_again), doneMessage = context.getString(R.string.actions_toast_unhidden),
        load = { api.blocklist() }, action = { api.setBlocklisted(false, it.type, it.tmdbId) }, onBack = onBack, onOpen = onOpen,
    )
}

@OptIn(ExperimentalMaterial3Api::class, ExperimentalFoundationApi::class)
@Composable
private fun TitleGridScreen(
    title: String,
    hint: String?,
    emptyMessage: String,
    actionTitle: String,
    doneMessage: String,
    load: suspend () -> List<MediaItem>,
    action: suspend (MediaItem) -> Unit,
    onBack: () -> Unit,
    onOpen: (MediaItem) -> Unit,
) {
    val context = LocalContext.current
    val toast = LocalToast.current
    val scope = rememberCoroutineScope()
    var items by remember { mutableStateOf<List<MediaItem>>(emptyList()) }
    var error by remember { mutableStateOf<ApiError?>(null) }
    var loaded by remember { mutableStateOf(false) }
    var refreshing by remember { mutableStateOf(false) }
    var menuFor by remember { mutableStateOf<String?>(null) }

    suspend fun reload() {
        try { items = load(); error = null } catch (e: ApiError) { if (items.isEmpty()) error = e }
        loaded = true
    }
    LaunchedEffect(Unit) { reload() }

    TopBarScaffold(title, onBack) {
        PullToRefreshBox(isRefreshing = refreshing, onRefresh = { scope.launch { refreshing = true; reload(); refreshing = false } }, modifier = Modifier.fillMaxSize()) {
            val failure = error
            when {
                failure != null -> if (failure == ApiError.Unreachable) OfflineState({ scope.launch { reload() } })
                else StateView(Icons.Outlined.WarningAmber, failure.message(context), actionTitle = stringResource(R.string.common_retry), onAction = { scope.launch { reload() } })
                loaded && items.isEmpty() -> StateView(Icons.Outlined.Inbox, emptyMessage, Modifier.height(320.dp))
                else -> LazyVerticalGrid(
                    GridCells.Adaptive(104.dp), Modifier.fillMaxSize(), contentPadding = PaddingValues(Tokens.Spacing.lg),
                    verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.lg), horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.md),
                ) {
                    if (hint != null) item(span = { GridItemSpan(maxLineSpan) }) { Text(hint, fontSize = Tokens.FontSize.subhead, color = Tokens.palette.mutedFg) }
                    if (!loaded) items(6) { PosterSkeleton() }
                    items(items, key = { it.id }) { item ->
                        Box {
                            MediaPoster(item, { onOpen(item) }, Modifier.combinedClickable(onClick = { onOpen(item) }, onLongClick = { menuFor = item.id }), longPressMenu = false)
                            DropdownMenu(menuFor == item.id, { menuFor = null }, containerColor = Tokens.palette.card) {
                                DropdownMenuItem(text = { Text(actionTitle, color = Tokens.palette.fg) }, onClick = {
                                    menuFor = null
                                    scope.launch {
                                        try { action(item); items = items.filter { it.id != item.id }; toast?.show(doneMessage) } catch (e: ApiError) { toast?.show(e.message(context), ToastKind.Error) }
                                    }
                                })
                            }
                        }
                    }
                }
            }
        }
    }
}
