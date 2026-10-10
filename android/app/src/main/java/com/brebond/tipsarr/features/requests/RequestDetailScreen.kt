package com.brebond.tipsarr.features.requests

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.Cancel
import androidx.compose.material.icons.outlined.Check
import androidx.compose.material.icons.outlined.Circle
import androidx.compose.material.icons.outlined.Close
import androidx.compose.material.icons.outlined.Movie
import androidx.compose.material.icons.outlined.Refresh
import androidx.compose.material.icons.outlined.RadioButtonChecked
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.heading
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import com.brebond.tipsarr.R
import com.brebond.tipsarr.core.api.ApiError
import com.brebond.tipsarr.core.api.MediaType
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
import com.brebond.tipsarr.features.detail.MediaRoute
import com.brebond.tipsarr.ui.Banner
import com.brebond.tipsarr.ui.BannerKind
import com.brebond.tipsarr.ui.ButtonKind
import com.brebond.tipsarr.ui.ProgressBar
import com.brebond.tipsarr.ui.RequestState
import com.brebond.tipsarr.ui.StatusBadge
import com.brebond.tipsarr.ui.TipsarrButton
import com.brebond.tipsarr.ui.TopBarScaffold
import kotlinx.coroutines.launch
import java.text.DateFormat
import java.util.Date

/** Detail of one request: progress, timeline, who asked, seasons, and the actions the person may take. */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun RequestDetailScreen(
    initial: RequestRecord,
    model: RequestsModel,
    currentUserName: String,
    onBack: () -> Unit,
    onOpenTitle: (MediaRoute) -> Unit,
) {
    val context = LocalContext.current
    val toast = LocalToast.current
    val scope = rememberCoroutineScope()
    var record by remember { mutableStateOf(initial) }
    var declining by remember { mutableStateOf(false) }
    var confirmDelete by remember { mutableStateOf(false) }
    var refreshing by remember { mutableStateOf(false) }
    val busy = record.id in model.busy

    suspend fun reload() { model.fresh(record.id)?.let { record = it } }
    fun run(work: suspend () -> Unit) {
        scope.launch { try { work() } catch (e: ApiError) { toast?.show(e.message(context), ToastKind.Error) } }
    }
    LaunchedEffect(initial.id) { reload() }
    val live = LocalLive.current
    OnTick(live?.requestsTick ?: 0) { reload() }

    TopBarScaffold(stringResource(R.string.m_tab_requests), onBack) {
        PullToRefreshBox(
            isRefreshing = refreshing,
            onRefresh = { scope.launch { refreshing = true; reload(); refreshing = false } },
            modifier = Modifier.fillMaxSize(),
        ) {
            Box(Modifier.fillMaxSize().verticalScroll(rememberScrollState()).navigationBarsPadding(), contentAlignment = Alignment.TopCenter) {
                Column(
                    Modifier.widthIn(max = 720.dp).fillMaxWidth().padding(Tokens.Spacing.lg),
                    verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.xl),
                ) {
                    Header(record)
                    val shown = record.withLive(live)
                    val percent = shown.progressPercent
                    if (record.state == RequestState.Downloading && percent != null) {
                        Column(verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm)) {
                            ProgressBar(percent)
                            Text(downloadText(context, percent, shown.etaSeconds), fontSize = Tokens.FontSize.footnote, color = Tokens.palette.mutedFg)
                        }
                    }
                    when (record.state) {
                        RequestState.Declined -> Banner(
                            record.declineReason?.let { stringResource(R.string.req_reason, it) } ?: stringResource(R.string.m_detail_declined),
                            kind = BannerKind.Error,
                        )
                        RequestState.Failed -> Banner(stringResource(R.string.req_failed_generic), kind = BannerKind.Warning, message = if (model.isAdmin) record.error else null)
                        else -> {}
                    }
                    if (model.isAdmin && !record.requestedBy.isNullOrEmpty()) Requester(record)
                    Column(verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.md)) {
                        SectionTitle(stringResource(R.string.m_requests_progress))
                        Timeline(steps(context, record))
                    }
                    Facts(record, currentUserName)
                    Actions(
                        record, model, busy,
                        onApprove = { run { record = model.approve(record) } },
                        onDecline = { declining = true },
                        onRetry = { run { record = model.retry(record) } },
                        onOpenTitle = { onOpenTitle(MediaRoute(record.type, record.tmdbId, record.title)) },
                        onDelete = { confirmDelete = true },
                    )
                }
            }
        }
    }

    if (declining) DeclineSheet(record.title, onDismiss = { declining = false }) { reason ->
        record = model.decline(record, reason)
        declining = false
    }
    if (confirmDelete) {
        AlertDialog(
            onDismissRequest = { confirmDelete = false },
            title = { Text(stringResource(R.string.m_request_cancel_confirm, record.title)) },
            confirmButton = {
                TextButton({ confirmDelete = false; run { model.delete(record); onBack() } }) { Text(stringResource(R.string.m_request_cancel), color = Tokens.palette.destructive) }
            },
            dismissButton = { TextButton({ confirmDelete = false }) { Text(stringResource(R.string.m_request_keep)) } },
            containerColor = Tokens.palette.card,
        )
    }
}

@Composable
private fun SectionTitle(text: String) {
    Text(text, fontSize = Tokens.FontSize.title3, fontWeight = FontWeight.Bold, color = Tokens.palette.fg, modifier = Modifier.semantics { heading() })
}

@Composable
private fun Header(record: RequestRecord) {
    Row(horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.lg), verticalAlignment = Alignment.Top) {
        Box(Modifier.width(96.dp).height(96.dp * Tokens.Size.posterRatio).clip(RoundedCornerShape(Tokens.Radius.md)).background(Tokens.palette.muted), contentAlignment = Alignment.Center) {
            Icon(Icons.Outlined.Movie, null, tint = Tokens.palette.mutedFg)
            RemoteImage(record.posterPath, TmdbSize.W342, Modifier.fillMaxSize())
        }
        Column(verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm)) {
            Text(record.title, fontSize = Tokens.FontSize.title3, fontWeight = FontWeight.Bold, color = Tokens.palette.fg, modifier = Modifier.semantics { heading() })
            Text(stringResource(if (record.type == MediaType.Tv) R.string.type_tv else R.string.type_movie), fontSize = Tokens.FontSize.footnote, color = Tokens.palette.mutedFg)
            StatusBadge(record.state)
        }
    }
}

@Composable
private fun Requester(record: RequestRecord) {
    val context = LocalContext.current
    Column(verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm)) {
        Text(stringResource(R.string.m_requests_requested_by), fontSize = Tokens.FontSize.footnote, fontWeight = FontWeight.SemiBold, color = Tokens.palette.mutedFg)
        Row(horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.md), verticalAlignment = Alignment.CenterVertically) {
            Box(Modifier.size(36.dp).clip(CircleShape).background(Tokens.palette.muted), contentAlignment = Alignment.Center) {
                Text(record.requestedBy.orEmpty().take(1).uppercase(), fontSize = Tokens.FontSize.headline, color = Tokens.palette.fg)
            }
            Column {
                Text(record.requestedBy.orEmpty(), fontSize = Tokens.FontSize.body, fontWeight = FontWeight.SemiBold, color = Tokens.palette.fg)
                Text(relativeTime(context, record.createdAt), fontSize = Tokens.FontSize.footnote, color = Tokens.palette.mutedFg)
            }
        }
    }
}

@Composable
private fun Facts(record: RequestRecord, currentUserName: String) {
    val context = LocalContext.current
    Column(verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.md)) {
        SectionTitle(stringResource(R.string.m_requests_details))
        Column {
            record.requestedBy?.takeIf { it.isNotEmpty() }?.let { name ->
                val me = name == currentUserName
                val you = stringResource(R.string.common_you).trim('(', ')').replaceFirstChar { it.titlecase() }
                Fact(stringResource(R.string.m_requests_requested_by), if (me) you else name)
            }
            if (record.seasons.isNotEmpty()) Fact(stringResource(R.string.fact_seasons), record.seasons.sorted().joinToString(", "))
        }
        if (record.seasons.isNotEmpty()) {
            Column(verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm), modifier = Modifier.padding(top = Tokens.Spacing.sm)) {
                record.seasons.sorted().forEach { number ->
                    val percent = record.seasonProgress[number]
                    Row(horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.md), verticalAlignment = Alignment.CenterVertically) {
                        Text(stringResource(R.string.req_season_badge, number.toString()), fontSize = Tokens.FontSize.subhead, fontWeight = FontWeight.SemiBold, color = Tokens.palette.fg)
                        Box(Modifier.weight(1f)) { ProgressBar(percent ?: 0, tint = if (percent == 100) Tokens.Status.available else Tokens.Status.downloading) }
                        Text(percent?.let { "$it%" } ?: "–", fontSize = Tokens.FontSize.footnote, color = Tokens.palette.mutedFg, textAlign = TextAlign.End, modifier = Modifier.width(44.dp))
                    }
                }
            }
        }
    }
}

@Composable
private fun Fact(label: String, value: String) {
    Column {
        Row(Modifier.fillMaxWidth().padding(vertical = Tokens.Spacing.sm)) {
            Text(label, fontSize = Tokens.FontSize.subhead, color = Tokens.palette.mutedFg)
            Spacer(Modifier.width(Tokens.Spacing.lg))
            Text(value, fontSize = Tokens.FontSize.subhead, color = Tokens.palette.fg, textAlign = TextAlign.End, modifier = Modifier.weight(1f))
        }
        HorizontalDivider(color = Tokens.palette.border)
    }
}

@Composable
private fun Actions(
    record: RequestRecord,
    model: RequestsModel,
    busy: Boolean,
    onApprove: () -> Unit,
    onDecline: () -> Unit,
    onRetry: () -> Unit,
    onOpenTitle: () -> Unit,
    onDelete: () -> Unit,
) {
    Column(verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm)) {
        if (model.isAdmin && record.state == RequestState.Requested) {
            Row(horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.md)) {
                Box(Modifier.weight(1f)) {
                    TipsarrButton(stringResource(R.string.req_approve), onApprove, fullWidth = true, enabled = !busy) { Icon(Icons.Outlined.Check, null, Modifier.size(20.dp)); Spacer(Modifier.width(Tokens.Spacing.sm)) }
                }
                Box(Modifier.weight(1f)) {
                    TipsarrButton(stringResource(R.string.req_decline), onDecline, fullWidth = true, kind = ButtonKind.Secondary, enabled = !busy) { Icon(Icons.Outlined.Close, null, Modifier.size(20.dp)); Spacer(Modifier.width(Tokens.Spacing.sm)) }
                }
            }
        }
        if (model.isAdmin && record.state == RequestState.Failed) {
            TipsarrButton(stringResource(R.string.req_retry), onRetry, fullWidth = true, enabled = !busy) { Icon(Icons.Outlined.Refresh, null, Modifier.size(20.dp)); Spacer(Modifier.width(Tokens.Spacing.sm)) }
        }
        TipsarrButton(stringResource(R.string.media_view_details), onOpenTitle, fullWidth = true, kind = ButtonKind.Secondary)
        if (model.canDelete(record)) {
            TipsarrButton(stringResource(R.string.m_request_cancel), onDelete, fullWidth = true, kind = ButtonKind.Ghost, enabled = !busy)
        }
    }
}

// Timeline

private enum class StepStatus { Done, Current, Todo, Failed }

private data class Step(val title: Int, val detail: String, val status: StepStatus)

private fun steps(context: android.content.Context, record: RequestRecord): List<Step> {
    val created = DateFormat.getDateTimeInstance(DateFormat.MEDIUM, DateFormat.SHORT).format(Date(record.createdAt * 1000))
    val decider = record.decidedBy?.let { context.getString(R.string.m_request_approved_by, it) }.orEmpty()
    return when (record.state) {
        RequestState.Declined -> listOf(
            Step(RequestState.Requested.title, created, StepStatus.Done),
            Step(RequestState.Declined.title, decider, StepStatus.Failed),
        )
        RequestState.Failed -> listOf(
            Step(RequestState.Requested.title, created, StepStatus.Done),
            Step(RequestState.Approved.title, decider, StepStatus.Done),
            Step(RequestState.Failed.title, "", StepStatus.Failed),
        )
        else -> {
            val order = listOf(RequestState.Requested, RequestState.Approved, RequestState.Searching, RequestState.Downloading, RequestState.Available)
            val current = order.indexOf(if (record.state == RequestState.Partial) RequestState.Available else record.state).coerceAtLeast(0)
            order.mapIndexed { index, state ->
                val detail = when (state) {
                    RequestState.Requested -> created
                    RequestState.Approved -> decider
                    RequestState.Downloading -> if (index <= current) record.progressPercent?.let { "$it%" }.orEmpty() else ""
                    else -> ""
                }
                val status = when {
                    record.state == RequestState.Available || index < current -> StepStatus.Done
                    index == current -> StepStatus.Current
                    else -> StepStatus.Todo
                }
                Step(state.title, detail, status)
            }
        }
    }
}

/** Vertical steps: done (green check), current, todo, failed. */
@Composable
private fun Timeline(steps: List<Step>) {
    Column {
        steps.forEachIndexed { index, step ->
            Row(horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.md), verticalAlignment = Alignment.Top) {
                Column(horizontalAlignment = Alignment.CenterHorizontally) {
                    val (icon, tint) = when (step.status) {
                        StepStatus.Done -> Icons.Filled.CheckCircle to Tokens.Status.available
                        StepStatus.Failed -> Icons.Filled.Cancel to Tokens.Status.failed
                        StepStatus.Current -> Icons.Outlined.RadioButtonChecked to Tokens.Status.downloading
                        StepStatus.Todo -> Icons.Outlined.Circle to Tokens.palette.border
                    }
                    Icon(icon, null, tint = tint, modifier = Modifier.size(28.dp))
                    if (index < steps.size - 1) Box(Modifier.width(2.dp).height(32.dp).background(if (step.status == StepStatus.Done) Tokens.Status.available else Tokens.palette.border))
                }
                Column(Modifier.padding(top = 2.dp)) {
                    Text(
                        stringResource(step.title), fontSize = Tokens.FontSize.body,
                        fontWeight = if (step.status == StepStatus.Todo) FontWeight.Medium else FontWeight.Bold,
                        color = if (step.status == StepStatus.Todo) Tokens.palette.mutedFg else Tokens.palette.fg,
                    )
                    if (step.detail.isNotEmpty()) Text(step.detail, fontSize = Tokens.FontSize.footnote, color = Tokens.palette.mutedFg)
                }
            }
        }
    }
}
