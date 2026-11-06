package com.brebond.tipsarr.features.teasers

import androidx.compose.foundation.ExperimentalFoundationApi
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.itemsIndexed
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.automirrored.outlined.Undo
import androidx.compose.material.icons.outlined.History
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.SwipeToDismissBox
import androidx.compose.material3.SwipeToDismissBoxValue
import androidx.compose.material3.rememberSwipeToDismissBoxState
import androidx.compose.ui.Alignment
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.RectangleShape
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import com.brebond.tipsarr.core.images.RemoteImage
import com.brebond.tipsarr.core.images.TmdbSize
import com.brebond.tipsarr.core.support.relativeTime
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
import com.brebond.tipsarr.core.api.adminForgetAllTeasers
import com.brebond.tipsarr.core.api.adminForgetTeaser
import com.brebond.tipsarr.core.api.adminTeaserHistory
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

/** The person whose history an admin manages (`/admin/users/{id}/teasers/...`). */
data class TeasersHistoryOwner(val id: String, val name: String)

/**
 * Titles swiped past in Teasers, newest first. Long press forgets one; the top bar wipes all.
 * With [owner] set, it is that person's history, managed by an admin.
 */
@OptIn(ExperimentalMaterial3Api::class, ExperimentalFoundationApi::class)
@Composable
fun TeasersHistoryScreen(api: TipsarrApi, onBack: () -> Unit, onOpen: (MediaItem) -> Unit, owner: TeasersHistoryOwner? = null) {
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
    val listState = rememberLazyListState()

    suspend fun reload() {
        try {
            val page = if (owner == null) api.teaserHistory(PAGE, 0) else api.adminTeaserHistory(owner.id, PAGE, 0)
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
            val page = if (owner == null) api.teaserHistory(PAGE, items.size) else api.adminTeaserHistory(owner.id, PAGE, items.size)
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
    LaunchedEffect(listState) {
        snapshotFlow { (listState.layoutInfo.visibleItemsInfo.lastOrNull()?.index ?: 0) to listState.layoutInfo.totalItemsCount }
            .distinctUntilChanged()
            .collect { (last, count) -> if (count > 0 && last >= count - 8) loadMore() }
    }

    TopBarScaffold(
        if (owner == null) stringResource(R.string.m_teasers_history_title) else stringResource(R.string.m_users_teasers_history_title, owner.name), onBack,
        actions = {
            // Like iOS: a text button "Wipe all" (destructive), not an icon.
            if (items.isNotEmpty()) TextButton({ confirmAll = true }) {
                Text(stringResource(R.string.m_teasers_history_forget_all), color = Tokens.palette.destructive)
            }
        },
    ) {
        PullToRefreshBox(isRefreshing = refreshing, onRefresh = { scope.launch { refreshing = true; reload(); refreshing = false } }, modifier = Modifier.fillMaxSize()) {
            val failure = error
            when {
                failure != null -> if (failure == ApiError.Unreachable) OfflineState({ scope.launch { reload() } })
                else StateView(Icons.Outlined.WarningAmber, failure.message(context), actionTitle = stringResource(R.string.common_retry), onAction = { scope.launch { reload() } })
                !loaded -> Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) { CircularProgressIndicator() }
                items.isEmpty() -> StateView(Icons.Outlined.History, stringResource(R.string.m_teasers_history_empty), Modifier.heightIn(min = 320.dp))
                else -> LazyColumn(Modifier.fillMaxSize(), listState, contentPadding = PaddingValues(Tokens.Spacing.lg)) {
                    item {
                        Text(
                            stringResource(R.string.m_teasers_history_hint), fontSize = Tokens.FontSize.footnote, color = Tokens.palette.mutedFg,
                            modifier = Modifier.padding(horizontal = Tokens.Spacing.xs, vertical = Tokens.Spacing.sm),
                        )
                    }
                    itemsIndexed(items, key = { _, it -> it.id }) { index, entry ->
                        val forgot = stringResource(R.string.m_teasers_history_forgot, entry.title)
                        val shape = when {
                            items.size == 1 -> RoundedCornerShape(Tokens.Radius.md)
                            index == 0 -> RoundedCornerShape(topStart = Tokens.Radius.md, topEnd = Tokens.Radius.md)
                            index == items.lastIndex -> RoundedCornerShape(bottomStart = Tokens.Radius.md, bottomEnd = Tokens.Radius.md)
                            else -> RectangleShape
                        }
                        // Swipe left to forget (iOS: trailing swipe action); the row springs back until the list drops it.
                        val dismiss = rememberSwipeToDismissBoxState(confirmValueChange = { value ->
                            if (value == SwipeToDismissBoxValue.EndToStart) {
                                scope.launch {
                                    try {
                                        if (owner == null) api.forgetTeaser(entry.type, entry.tmdbId) else api.adminForgetTeaser(owner.id, entry.type, entry.tmdbId)
                                        items = items.filter { it.id != entry.id }
                                        total = (total - 1).coerceAtLeast(0)
                                        toast?.show(forgot)
                                    } catch (e: ApiError) { toast?.show(e.message(context), ToastKind.Error) }
                                }
                            }
                            false
                        })
                        Column(Modifier.clip(shape)) {
                            SwipeToDismissBox(
                                dismiss, enableDismissFromStartToEnd = false,
                                backgroundContent = {
                                    Row(Modifier.fillMaxSize().background(Tokens.palette.primary).padding(horizontal = Tokens.Spacing.lg), horizontalArrangement = Arrangement.End, verticalAlignment = Alignment.CenterVertically) {
                                        Icon(Icons.AutoMirrored.Outlined.Undo, null, tint = Tokens.palette.primaryFg)
                                        Text(stringResource(R.string.m_teasers_history_forget), color = Tokens.palette.primaryFg, fontSize = Tokens.FontSize.subhead, modifier = Modifier.padding(start = Tokens.Spacing.sm))
                                    }
                                },
                            ) {
                                Row(
                                    Modifier.fillMaxWidth().background(Tokens.palette.card).clickable { onOpen(entry.item) }.padding(horizontal = Tokens.Spacing.lg, vertical = Tokens.Spacing.sm),
                                    horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.md), verticalAlignment = Alignment.CenterVertically,
                                ) {
                                    Box(Modifier.width(44.dp).height(66.dp).clip(RoundedCornerShape(Tokens.Radius.sm)).background(Tokens.palette.muted)) {
                                        RemoteImage(entry.posterPath, TmdbSize.W185, Modifier.fillMaxSize())
                                    }
                                    Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
                                        Text(entry.title, fontSize = Tokens.FontSize.body, fontWeight = FontWeight.Medium, color = Tokens.palette.fg, maxLines = 2, overflow = TextOverflow.Ellipsis)
                                        Text(
                                            stringResource(R.string.m_teasers_history_seen_on, relativeTime(context, entry.seenAt)),
                                            fontSize = Tokens.FontSize.footnote, color = Tokens.palette.mutedFg,
                                        )
                                    }
                                }
                            }
                            if (index < items.lastIndex) HorizontalDivider(Modifier.padding(start = 76.dp), color = Tokens.palette.border)
                        }
                    }
                }
            }
        }
    }

    if (confirmAll) {
        val wiped = if (owner == null) stringResource(R.string.m_teasers_history_wiped) else stringResource(R.string.m_users_teasers_history_wiped, owner.name)
        AlertDialog(
            onDismissRequest = { confirmAll = false },
            title = { Text(if (owner == null) stringResource(R.string.m_teasers_history_confirm_all) else stringResource(R.string.m_users_teasers_history_confirm_all, owner.name)) },
            confirmButton = {
                TextButton({
                    confirmAll = false
                    scope.launch {
                        try {
                            if (owner == null) api.forgetAllTeasers() else api.adminForgetAllTeasers(owner.id)
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
