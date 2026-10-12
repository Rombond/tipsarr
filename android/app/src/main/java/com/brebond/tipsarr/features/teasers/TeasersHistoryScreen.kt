package com.brebond.tipsarr.features.teasers

import androidx.compose.foundation.ExperimentalFoundationApi
import androidx.compose.foundation.combinedClickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.GridItemSpan
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.foundation.lazy.grid.rememberLazyGridState
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.DeleteSweep
import androidx.compose.material.icons.outlined.Inbox
import androidx.compose.material.icons.outlined.WarningAmber
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.runtime.snapshotFlow
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import com.brebond.tipsarr.R
import com.brebond.tipsarr.core.api.ApiError
import com.brebond.tipsarr.core.api.MediaItem
import com.brebond.tipsarr.core.api.TeaserHistoryEntry
import com.brebond.tipsarr.core.api.TipsarrApi
import com.brebond.tipsarr.core.api.forgetAllTeasers
import com.brebond.tipsarr.core.api.forgetTeaser
import com.brebond.tipsarr.core.api.teaserHistory
import com.brebond.tipsarr.core.support.LocalToast
import com.brebond.tipsarr.core.support.ToastKind
import com.brebond.tipsarr.design.Tokens
import com.brebond.tipsarr.design.palette
import com.brebond.tipsarr.ui.MediaPoster
import com.brebond.tipsarr.ui.OfflineState
import com.brebond.tipsarr.ui.PosterSkeleton
import com.brebond.tipsarr.ui.StateView
import com.brebond.tipsarr.ui.TopBarScaffold
import kotlinx.coroutines.flow.distinctUntilChanged
import kotlinx.coroutines.launch
import java.text.DateFormat
import java.util.Date

private const val PAGE = 40

/** Titles swiped past in Teasers, newest first. Long press forgets one; the top bar wipes all. */
@OptIn(ExperimentalMaterial3Api::class, ExperimentalFoundationApi::class)
@Composable
fun TeasersHistoryScreen(api: TipsarrApi, onBack: () -> Unit, onOpen: (MediaItem) -> Unit) {
    val context = LocalContext.current
    val toast = LocalToast.current
    val scope = rememberCoroutineScope()
    var items by remember { mutableStateOf<List<TeaserHistoryEntry>>(emptyList()) }
    var total by remember { mutableStateOf(0) }
    var error by remember { mutableStateOf<ApiError?>(null) }
    var loaded by remember { mutableStateOf(false) }
    var refreshing by remember { mutableStateOf(false) }
    var loadingMore by remember { mutableStateOf(false) }
    var menuFor by remember { mutableStateOf<String?>(null) }
    var confirmAll by remember { mutableStateOf(false) }
    val dates = remember { DateFormat.getDateInstance(DateFormat.MEDIUM) }
    val gridState = rememberLazyGridState()

    suspend fun reload() {
        try {
            val page = api.teaserHistory(PAGE, 0)
            items = page.items
            total = page.total
            error = null
        } catch (e: ApiError) { if (items.isEmpty()) error = e }
        loaded = true
    }
    suspend fun loadMore() {
        if (!loaded || loadingMore || items.isEmpty() || items.size >= total) return
        loadingMore = true
        try {
            val page = api.teaserHistory(PAGE, items.size)
            val known = items.map { it.id }.toSet()
            items = items + page.items.filter { it.id !in known }
            total = page.total
        } catch (_: ApiError) {
            // tried again on the next scroll
        } finally {
            loadingMore = false
        }
    }
    LaunchedEffect(Unit) { reload() }
    // Near the end of the list: the next page.
    LaunchedEffect(gridState) {
        snapshotFlow { (gridState.layoutInfo.visibleItemsInfo.lastOrNull()?.index ?: 0) to gridState.layoutInfo.totalItemsCount }
            .distinctUntilChanged()
            .collect { (last, count) -> if (count > 0 && last >= count - 8) loadMore() }
    }

    TopBarScaffold(
        stringResource(R.string.m_teasers_history_title), onBack,
        actions = {
            if (items.isNotEmpty()) IconButton({ confirmAll = true }) {
                Icon(Icons.Outlined.DeleteSweep, stringResource(R.string.m_teasers_history_forget_all), tint = Tokens.palette.fg)
            }
        },
    ) {
        PullToRefreshBox(isRefreshing = refreshing, onRefresh = { scope.launch { refreshing = true; reload(); refreshing = false } }, modifier = Modifier.fillMaxSize()) {
            val failure = error
            when {
                failure != null -> if (failure == ApiError.Unreachable) OfflineState({ scope.launch { reload() } })
                else StateView(Icons.Outlined.WarningAmber, failure.message(context), actionTitle = stringResource(R.string.common_retry), onAction = { scope.launch { reload() } })
                loaded && items.isEmpty() -> StateView(Icons.Outlined.Inbox, stringResource(R.string.m_teasers_history_empty), Modifier.height(320.dp))
                else -> LazyVerticalGrid(
                    GridCells.Adaptive(104.dp), Modifier.fillMaxSize(), gridState, contentPadding = PaddingValues(Tokens.Spacing.lg),
                    verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.lg), horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.md),
                ) {
                    item(span = { GridItemSpan(maxLineSpan) }) {
                        Text(stringResource(R.string.m_teasers_history_hint), fontSize = Tokens.FontSize.subhead, color = Tokens.palette.mutedFg)
                    }
                    if (!loaded) items(6) { PosterSkeleton() }
                    items(items, key = { it.id }) { entry ->
                        val seen = stringResource(R.string.m_teasers_history_seen_on, dates.format(Date(entry.seenAt * 1000)))
                        val forgot = stringResource(R.string.m_teasers_history_forgot, entry.title)
                        Box {
                            MediaPoster(
                                entry.item, { onOpen(entry.item) },
                                Modifier.combinedClickable(onClick = { onOpen(entry.item) }, onLongClick = { menuFor = entry.id }),
                                longPressMenu = false, subtitle = seen,
                            )
                            DropdownMenu(menuFor == entry.id, { menuFor = null }, containerColor = Tokens.palette.card) {
                                DropdownMenuItem(text = { Text(stringResource(R.string.m_teasers_history_forget), color = Tokens.palette.fg) }, onClick = {
                                    menuFor = null
                                    scope.launch {
                                        try {
                                            api.forgetTeaser(entry.type, entry.tmdbId)
                                            items = items.filter { it.id != entry.id }
                                            total = (total - 1).coerceAtLeast(0)
                                            toast?.show(forgot)
                                        } catch (e: ApiError) { toast?.show(e.message(context), ToastKind.Error) }
                                    }
                                })
                            }
                        }
                    }
                }
            }
        }
    }

    if (confirmAll) {
        val wiped = stringResource(R.string.m_teasers_history_wiped)
        AlertDialog(
            onDismissRequest = { confirmAll = false },
            title = { Text(stringResource(R.string.m_teasers_history_confirm_all)) },
            confirmButton = {
                TextButton({
                    confirmAll = false
                    scope.launch {
                        try {
                            api.forgetAllTeasers()
                            items = emptyList()
                            total = 0
                            toast?.show(wiped)
                        } catch (e: ApiError) { toast?.show(e.message(context), ToastKind.Error) }
                    }
                }) { Text(stringResource(R.string.m_teasers_history_forget_all), color = Tokens.palette.destructive) }
            },
            dismissButton = { TextButton({ confirmAll = false }) { Text(stringResource(R.string.common_cancel)) } },
            containerColor = Tokens.palette.card,
        )
    }
}
