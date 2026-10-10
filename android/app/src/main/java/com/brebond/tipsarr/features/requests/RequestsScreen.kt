package com.brebond.tipsarr.features.requests

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.itemsIndexed
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.Checklist
import androidx.compose.material.icons.outlined.Close
import androidx.compose.material.icons.outlined.Check
import androidx.compose.material.icons.outlined.Delete
import androidx.compose.material.icons.outlined.Movie
import androidx.compose.material.icons.outlined.WarningAmber
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.SwipeToDismissBox
import androidx.compose.material3.SwipeToDismissBoxValue
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.material3.rememberSwipeToDismissBoxState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.heading
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.brebond.tipsarr.R
import com.brebond.tipsarr.core.api.ApiError
import com.brebond.tipsarr.core.api.MediaType
import com.brebond.tipsarr.core.api.RequestFilter
import com.brebond.tipsarr.core.api.RequestRecord
import com.brebond.tipsarr.core.images.RemoteImage
import com.brebond.tipsarr.core.images.TmdbSize
import com.brebond.tipsarr.core.live.LocalLive
import com.brebond.tipsarr.core.live.OnTick
import com.brebond.tipsarr.core.support.LocalToast
import com.brebond.tipsarr.core.support.ToastKind
import com.brebond.tipsarr.core.support.relativeTime
import com.brebond.tipsarr.design.Tokens
import com.brebond.tipsarr.design.palette
import com.brebond.tipsarr.features.discover.Phase
import com.brebond.tipsarr.features.detail.formatEta
import com.brebond.tipsarr.ui.ButtonKind
import com.brebond.tipsarr.ui.Chip
import com.brebond.tipsarr.ui.OfflineState
import com.brebond.tipsarr.ui.ProgressBar
import com.brebond.tipsarr.ui.RequestState
import com.brebond.tipsarr.ui.Skeleton
import com.brebond.tipsarr.ui.SkeletonBlock
import com.brebond.tipsarr.ui.StateView
import com.brebond.tipsarr.ui.StatusBadge
import com.brebond.tipsarr.ui.TipsarrButton
import kotlinx.coroutines.launch

private val GUTTER = Tokens.Spacing.lg

private fun RequestFilter.title(): Int = when (this) {
    RequestFilter.All -> R.string.requests_tab_all
    RequestFilter.Pending -> R.string.requests_tab_pending
    RequestFilter.Approved -> R.string.requests_tab_approved
    RequestFilter.Available -> R.string.requests_tab_available
    RequestFilter.Declined -> R.string.requests_tab_declined
    RequestFilter.Failed -> R.string.requests_tab_failed
}

/** Requests tab: status chips with counts, then every request with its progress; admins approve or decline in the row. */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun RequestsScreen(model: RequestsModel, onOpen: (RequestRecord) -> Unit, onOpenDiscover: () -> Unit, modifier: Modifier = Modifier) {
    val context = LocalContext.current
    val toast = LocalToast.current
    val scope = rememberCoroutineScope()
    var refreshing by remember { mutableStateOf(false) }
    var declining by remember { mutableStateOf<RequestRecord?>(null) }
    var deleting by remember { mutableStateOf<RequestRecord?>(null) }
    LaunchedEffect(model.filter) { model.load(model.filter) }
    val live = LocalLive.current
    OnTick(live?.requestsTick ?: 0) { model.load() }

    fun run(work: suspend () -> Unit) {
        scope.launch { try { work() } catch (e: ApiError) { toast?.show(e.message(context), ToastKind.Error) } }
    }

    PullToRefreshBox(
        isRefreshing = refreshing,
        onRefresh = { scope.launch { refreshing = true; model.load(); refreshing = false } },
        modifier = modifier.fillMaxSize(),
    ) {
        LazyColumn(Modifier.fillMaxSize(), contentPadding = PaddingValues(bottom = Tokens.Spacing.lg)) {
            item {
                Column(verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.md), modifier = Modifier.padding(top = Tokens.Spacing.sm, bottom = Tokens.Spacing.sm)) {
                    Text(
                        stringResource(R.string.m_tab_requests), fontSize = Tokens.FontSize.largeTitle, fontWeight = FontWeight.Bold,
                        color = Tokens.palette.fg, modifier = Modifier.padding(horizontal = GUTTER).semantics { heading() },
                    )
                    LazyRow(horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm), contentPadding = PaddingValues(horizontal = GUTTER)) {
                        items(RequestFilter.entries) { item ->
                            Chip(
                                stringResource(item.title()), selected = model.filter == item, onClick = { model.filter = item },
                                count = if (item == RequestFilter.All) null else model.counts.count(item),
                            )
                        }
                    }
                }
            }
            when (val phase = model.phase) {
                Phase.Idle, Phase.Loading -> items(6) { RowSkeleton() }
                is Phase.Failed -> item {
                    Box(Modifier.fillMaxWidth().height(360.dp)) {
                        if (phase.error == ApiError.Unreachable) OfflineState({ scope.launch { model.load() } })
                        else StateView(Icons.Outlined.WarningAmber, phase.error.message(context), actionTitle = stringResource(R.string.common_retry), onAction = { scope.launch { model.load() } })
                    }
                }
                Phase.Loaded -> if (model.items.isEmpty()) item {
                    Box(Modifier.fillMaxWidth().height(360.dp)) {
                        if (model.filter == RequestFilter.All) {
                            StateView(
                                Icons.Outlined.Checklist, stringResource(R.string.requests_empty_all), message = stringResource(R.string.requests_empty_hint),
                                actionTitle = stringResource(R.string.m_requests_empty_cta), onAction = onOpenDiscover,
                            )
                        } else {
                            StateView(Icons.Outlined.Checklist, stringResource(R.string.requests_empty_tab, stringResource(model.filter.title())))
                        }
                    }
                } else {
                    itemsIndexed(model.items, key = { _, item -> item.id }) { index, record ->
                        LaunchedEffect(index, model.items.size) { model.loadMore(index) }
                        SwipeRow(record, model.canDelete(record), onDelete = { deleting = record }) {
                            RequestRow(
                                record.withLive(live), showRequester = model.isAdmin, busy = record.id in model.busy,
                                onClick = { onOpen(record) },
                                onApprove = { run { val updated = model.approve(record); toast?.show(approvedText(context, updated, record.title)) } },
                                onDecline = { declining = record },
                            )
                        }
                        HorizontalDivider(Modifier.padding(start = GUTTER), color = Tokens.palette.border)
                    }
                    if (model.loadingMore) item {
                        Box(Modifier.fillMaxWidth().padding(Tokens.Spacing.lg), contentAlignment = Alignment.Center) { CircularProgressIndicator() }
                    }
                }
            }
        }
    }

    declining?.let { record ->
        DeclineSheet(record.title, onDismiss = { declining = null }) { reason ->
            model.decline(record, reason)
            declining = null
        }
    }
    deleting?.let { record ->
        AlertDialog(
            onDismissRequest = { deleting = null },
            title = { Text(stringResource(R.string.m_request_cancel_confirm, record.title)) },
            confirmButton = { TextButton({ deleting = null; run { model.delete(record) } }) { Text(stringResource(R.string.m_request_cancel), color = Tokens.palette.destructive) } },
            dismissButton = { TextButton({ deleting = null }) { Text(stringResource(R.string.m_request_keep)) } },
            containerColor = Tokens.palette.card,
        )
    }
}

internal fun approvedText(context: android.content.Context, record: RequestRecord, title: String): String {
    val name = if (record.dryRun) "media_toast_approved_dry" else "media_toast_approved"
    val id = context.resources.getIdentifier(name, "string", context.packageName)
    return if (id == 0) title else context.getString(id, title)
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun SwipeRow(record: RequestRecord, canDelete: Boolean, onDelete: () -> Unit, content: @Composable () -> Unit) {
    if (!canDelete) { content(); return }
    val state = rememberSwipeToDismissBoxState(confirmValueChange = {
        if (it == SwipeToDismissBoxValue.EndToStart) onDelete()
        false // the row stays until the confirmation removes it
    })
    SwipeToDismissBox(
        state = state,
        enableDismissFromStartToEnd = false,
        backgroundContent = {
            Box(Modifier.fillMaxSize().background(Tokens.palette.destructive).padding(horizontal = Tokens.Spacing.xl), contentAlignment = Alignment.CenterEnd) {
                Icon(Icons.Outlined.Delete, stringResource(R.string.m_requests_cancel_swipe), tint = Tokens.palette.bg)
            }
        },
    ) { Box(Modifier.background(Tokens.palette.bg)) { content() } }
}

@Composable
private fun RowSkeleton() {
    Skeleton(Modifier.padding(horizontal = GUTTER, vertical = Tokens.Spacing.md)) {
        Row(horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.md)) {
            Box(Modifier.width(64.dp).height(96.dp).clip(RoundedCornerShape(Tokens.Radius.sm)).background(Tokens.palette.border))
            Column(verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm)) {
                SkeletonBlock(Modifier.width(180.dp), height = 16.dp)
                SkeletonBlock(Modifier.width(110.dp), height = 12.dp)
                SkeletonBlock(Modifier.width(140.dp), height = 12.dp)
            }
        }
    }
}

/** One request: poster, title, state, who and when; a bar while downloading; approve / decline for admins. */
@Composable
fun RequestRow(
    record: RequestRecord,
    showRequester: Boolean,
    busy: Boolean,
    onClick: () -> Unit,
    onApprove: () -> Unit,
    onDecline: () -> Unit,
) {
    val context = LocalContext.current
    val whenText = relativeTime(context, record.createdAt).let { ago ->
        val name = record.requestedBy
        if (showRequester && !name.isNullOrEmpty()) stringResource(R.string.m_requests_by_ago, name, ago) else stringResource(R.string.m_requests_requested_ago, ago)
    }
    Row(
        Modifier.fillMaxWidth().clickable(onClick = onClick).alpha(if (busy) 0.6f else 1f).padding(horizontal = GUTTER, vertical = Tokens.Spacing.md),
        horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.md), verticalAlignment = Alignment.Top,
    ) {
        Box(Modifier.width(64.dp).height(96.dp).clip(RoundedCornerShape(Tokens.Radius.sm)).background(Tokens.palette.muted), contentAlignment = Alignment.Center) {
            Icon(Icons.Outlined.Movie, null, tint = Tokens.palette.mutedFg)
            RemoteImage(record.posterPath, TmdbSize.W185, Modifier.fillMaxSize())
        }
        Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.xs)) {
            Row(horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm), verticalAlignment = Alignment.Top) {
                Text(record.title, fontSize = Tokens.FontSize.body, fontWeight = FontWeight.SemiBold, color = Tokens.palette.fg, maxLines = 2, overflow = TextOverflow.Ellipsis, modifier = Modifier.weight(1f))
                StatusBadge(record.state)
            }
            Text(stringResource(if (record.type == MediaType.Tv) R.string.type_tv else R.string.type_movie), fontSize = Tokens.FontSize.footnote, color = Tokens.palette.mutedFg)
            Text(whenText, fontSize = Tokens.FontSize.caption, color = Tokens.palette.mutedFg)
            val percent = record.progressPercent
            if (record.state == RequestState.Downloading && percent != null) {
                ProgressBar(percent, Modifier.padding(top = Tokens.Spacing.xs))
                Text(downloadText(context, percent, record.etaSeconds), fontSize = Tokens.FontSize.caption, color = Tokens.palette.mutedFg)
            }
            if (showRequester && record.state == RequestState.Requested) {
                Row(horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm), modifier = Modifier.padding(top = Tokens.Spacing.xs)) {
                    TipsarrButton(stringResource(R.string.req_approve), onApprove, compact = true, enabled = !busy) { Icon(Icons.Outlined.Check, null, Modifier.size(16.dp)); Spacer(Modifier.width(Tokens.Spacing.xs)) }
                    TipsarrButton(stringResource(R.string.req_decline), onDecline, compact = true, kind = ButtonKind.Secondary, enabled = !busy) { Icon(Icons.Outlined.Close, null, Modifier.size(16.dp)); Spacer(Modifier.width(Tokens.Spacing.xs)) }
                }
            }
        }
    }
}

/** "Downloading 45% · 12m left", as on the web. */
internal fun downloadText(context: android.content.Context, percent: Int, etaSeconds: Int?): String {
    val eta = etaSeconds?.takeIf { it > 0 }
    val pct = context.getString(R.string.req_stage_downloading_pct, percent.toString())
    return if (eta == null) pct else "$pct · ${formatEta(eta)}"
}

/** The record with the newer progress the server pushed, when there is one. */
internal fun RequestRecord.withLive(live: com.brebond.tipsarr.core.live.LiveUpdates?): RequestRecord {
    val pushed = live?.progress?.get(id) ?: return this
    return copy(progressPercent = pushed.percent, etaSeconds = pushed.etaSeconds)
}
