package com.brebond.tipsarr.features.admin

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
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.itemsIndexed
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.outlined.KeyboardArrowRight
import androidx.compose.material.icons.automirrored.outlined.Send
import androidx.compose.material.icons.outlined.ChatBubbleOutline
import androidx.compose.material.icons.outlined.CheckCircle
import androidx.compose.material.icons.outlined.Movie
import androidx.compose.material.icons.outlined.Undo
import androidx.compose.material.icons.outlined.Verified
import androidx.compose.material.icons.outlined.WarningAmber
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.OutlinedTextFieldDefaults
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
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.heading
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardCapitalization
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.brebond.tipsarr.R
import com.brebond.tipsarr.core.api.ApiError
import com.brebond.tipsarr.core.api.IssueFilter
import com.brebond.tipsarr.core.api.IssueKind
import com.brebond.tipsarr.core.api.IssueThread
import com.brebond.tipsarr.core.api.IssueView
import com.brebond.tipsarr.core.api.TipsarrApi
import com.brebond.tipsarr.core.api.comment
import com.brebond.tipsarr.core.api.deleteIssue
import com.brebond.tipsarr.core.api.issue
import com.brebond.tipsarr.core.api.issues
import com.brebond.tipsarr.core.api.openIssueCount
import com.brebond.tipsarr.core.api.reopenIssue
import com.brebond.tipsarr.core.api.resolveIssue
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
import com.brebond.tipsarr.features.discover.Phase
import com.brebond.tipsarr.features.profile.AvatarView
import com.brebond.tipsarr.ui.ButtonKind
import com.brebond.tipsarr.ui.Chip
import com.brebond.tipsarr.ui.OfflineState
import com.brebond.tipsarr.ui.Skeleton
import com.brebond.tipsarr.ui.SkeletonBlock
import com.brebond.tipsarr.ui.StateView
import com.brebond.tipsarr.ui.TipsarrButton
import com.brebond.tipsarr.ui.TopBarScaffold
import kotlinx.coroutines.launch

private fun IssueFilter.title(): Int = when (this) {
    IssueFilter.Open -> R.string.issue_filter_open
    IssueFilter.Resolved -> R.string.issue_filter_resolved
    IssueFilter.All -> R.string.issue_filter_all
}

private fun IssueKind.title(): Int = when (this) {
    IssueKind.video -> R.string.issue_kind_video
    IssueKind.audio -> R.string.issue_kind_audio
    IssueKind.subtitles -> R.string.issue_kind_subtitles
    IssueKind.other -> R.string.issue_kind_other
}

/** "S2 · E4", "S2" or null for the whole title. */
@Composable
private fun scopeText(season: Int?, episode: Int?): String? = when {
    season != null && episode != null -> stringResource(R.string.issue_s_e, season.toString(), episode.toString())
    season != null -> "S$season"
    else -> null
}

/** Open and resolved problem reports (admins see everyone's). */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun IssuesScreen(api: TipsarrApi, onBack: () -> Unit, onOpen: (String) -> Unit) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    var filter by remember { mutableStateOf(IssueFilter.Open) }
    var items by remember { mutableStateOf<List<IssueView>>(emptyList()) }
    var total by remember { mutableStateOf(0) }
    var openCount by remember { mutableStateOf(0) }
    var phase by remember { mutableStateOf<Phase>(Phase.Idle) }
    var loadingMore by remember { mutableStateOf(false) }
    var refreshing by remember { mutableStateOf(false) }

    suspend fun load() {
        val current = filter
        if (items.isEmpty()) phase = Phase.Loading
        try {
            val page = api.issues(current, 0)
            if (current != filter) return
            items = page.items; total = page.total; phase = Phase.Loaded
        } catch (e: ApiError) {
            if (current == filter && items.isEmpty()) phase = Phase.Failed(e)
        }
        runCatching { api.openIssueCount() }.getOrNull()?.let { openCount = it }
    }
    LaunchedEffect(filter) { load() }
    val live = LocalLive.current
    OnTick(live?.issuesTick ?: 0) { load() }

    TopBarScaffold(stringResource(R.string.issue_title), onBack) {
        PullToRefreshBox(isRefreshing = refreshing, onRefresh = { scope.launch { refreshing = true; load(); refreshing = false } }, modifier = Modifier.fillMaxSize()) {
            LazyColumn(Modifier.fillMaxSize(), contentPadding = PaddingValues(bottom = Tokens.Spacing.lg)) {
                item {
                    LazyRow(horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm), contentPadding = PaddingValues(Tokens.Spacing.lg)) {
                        items(IssueFilter.entries) { item ->
                            Chip(stringResource(item.title()), selected = filter == item, onClick = { items = emptyList(); filter = item }, count = if (item == IssueFilter.Open) openCount else null)
                        }
                    }
                }
                when (val current = phase) {
                    Phase.Idle, Phase.Loading -> items(5) { RowSkeleton() }
                    is Phase.Failed -> item {
                        Box(Modifier.fillMaxWidth().height(360.dp)) {
                            if (current.error == ApiError.Unreachable) OfflineState({ scope.launch { load() } })
                            else StateView(Icons.Outlined.WarningAmber, current.error.message(context), actionTitle = stringResource(R.string.common_retry), onAction = { scope.launch { load() } })
                        }
                    }
                    Phase.Loaded -> if (items.isEmpty()) item {
                        StateView(Icons.Outlined.Verified, stringResource(R.string.issue_empty), Modifier.height(360.dp))
                    } else {
                        itemsIndexed(items, key = { _, issue -> issue.id }) { index, issue ->
                            LaunchedEffect(index, items.size) {
                                if (!loadingMore && items.size < total && index >= items.size - 4) {
                                    loadingMore = true
                                    val current = filter
                                    runCatching { api.issues(current, items.size) }.getOrNull()?.let { page ->
                                        if (current == filter) { val known = items.mapTo(HashSet()) { it.id }; items = items + page.items.filter { it.id !in known }; total = page.total }
                                    }
                                    loadingMore = false
                                }
                            }
                            IssueRow(issue) { onOpen(issue.id) }
                            HorizontalDivider(Modifier.padding(start = Tokens.Spacing.lg), color = Tokens.palette.border)
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun RowSkeleton() {
    Skeleton(Modifier.padding(horizontal = Tokens.Spacing.lg, vertical = Tokens.Spacing.md)) {
        Row(horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.md)) {
            Box(Modifier.width(48.dp).height(72.dp).clip(RoundedCornerShape(Tokens.Radius.sm)).background(Tokens.palette.border))
            Column(verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm)) {
                SkeletonBlock(Modifier.width(160.dp), height = 16.dp)
                SkeletonBlock(Modifier.width(110.dp), height = 12.dp)
            }
        }
    }
}

@Composable
private fun IssueRow(issue: IssueView, onClick: () -> Unit) {
    val context = LocalContext.current
    val scope = scopeText(issue.season, issue.episode)
    Row(
        Modifier.fillMaxWidth().clickable(onClick = onClick).padding(horizontal = Tokens.Spacing.lg, vertical = Tokens.Spacing.md),
        horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.md), verticalAlignment = Alignment.Top,
    ) {
        Box(Modifier.width(48.dp).height(72.dp).clip(RoundedCornerShape(Tokens.Radius.sm)).background(Tokens.palette.muted), contentAlignment = Alignment.Center) {
            Icon(Icons.Outlined.Movie, null, tint = Tokens.palette.mutedFg)
            RemoteImage(issue.posterPath, TmdbSize.W92, Modifier.fillMaxSize())
        }
        Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.xs)) {
            Text(issue.title, fontSize = Tokens.FontSize.body, fontWeight = FontWeight.SemiBold, color = Tokens.palette.fg, maxLines = 2, overflow = TextOverflow.Ellipsis)
            Row(horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm), verticalAlignment = Alignment.CenterVertically) {
                Pill(stringResource(issue.kind.title()))
                if (scope != null) Text(scope, fontSize = Tokens.FontSize.caption, color = Tokens.palette.mutedFg)
            }
            Text(
                stringResource(R.string.m_admin_reported_by, issue.createdBy.name) + " · " + relativeTime(context, issue.createdAt),
                fontSize = Tokens.FontSize.caption, color = Tokens.palette.mutedFg,
            )
        }
        Column(horizontalAlignment = Alignment.End, verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.xs)) {
            Text(
                stringResource(if (issue.isOpen) R.string.issue_open else R.string.issue_resolved), fontSize = Tokens.FontSize.caption, fontWeight = FontWeight.SemiBold,
                color = if (issue.isOpen) Tokens.Status.requested else Tokens.Status.available,
            )
            if (issue.commentCount > 0) {
                Row(horizontalArrangement = Arrangement.spacedBy(2.dp), verticalAlignment = Alignment.CenterVertically) {
                    Icon(Icons.Outlined.ChatBubbleOutline, null, tint = Tokens.palette.mutedFg, modifier = Modifier.size(14.dp))
                    Text(issue.commentCount.toString(), fontSize = Tokens.FontSize.caption, color = Tokens.palette.mutedFg)
                }
            }
        }
    }
}

@Composable
private fun Pill(text: String) {
    Text(text, fontSize = Tokens.FontSize.caption, fontWeight = FontWeight.SemiBold, color = Tokens.palette.fg,
        modifier = Modifier.clip(CircleShape).background(Tokens.palette.muted).padding(horizontal = Tokens.Spacing.sm, vertical = 2.dp))
}

/** One problem report: the title, its thread, a reply box and (admins) resolve or delete. */
@Composable
fun IssueDetailScreen(api: TipsarrApi, id: String, isAdmin: Boolean, onBack: () -> Unit, onOpenTitle: (MediaRoute) -> Unit) {
    val context = LocalContext.current
    val toast = LocalToast.current
    val scope = rememberCoroutineScope()
    var thread by remember { mutableStateOf<IssueThread?>(null) }
    var error by remember { mutableStateOf<ApiError?>(null) }
    var reply by remember { mutableStateOf("") }
    var sending by remember { mutableStateOf(false) }
    var confirmDelete by remember { mutableStateOf(false) }

    suspend fun load() { try { thread = api.issue(id); error = null } catch (e: ApiError) { if (thread == null) error = e } }
    fun run(work: suspend () -> Unit) { scope.launch { try { work() } catch (e: ApiError) { toast?.show(e.message(context), ToastKind.Error) } } }
    LaunchedEffect(id) { load() }
    val live = LocalLive.current
    OnTick(live?.issuesTick ?: 0) { load() }

    TopBarScaffold(stringResource(R.string.issue_title), onBack) {
        val current = thread
        val failure = error
        when {
            current != null -> Box(Modifier.fillMaxSize().imePadding().verticalScroll(rememberScrollState()).navigationBarsPadding(), contentAlignment = Alignment.TopCenter) {
                Column(Modifier.widthIn(max = 720.dp).fillMaxWidth().padding(Tokens.Spacing.lg), verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.xl)) {
                    Row(
                        Modifier.fillMaxWidth().clickable { onOpenTitle(MediaRoute(current.type, current.tmdbId, current.title)) },
                        horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.md), verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Box(Modifier.width(64.dp).height(96.dp).clip(RoundedCornerShape(Tokens.Radius.sm)).background(Tokens.palette.muted), contentAlignment = Alignment.Center) {
                            Icon(Icons.Outlined.Movie, null, tint = Tokens.palette.mutedFg)
                            RemoteImage(current.posterPath, TmdbSize.W185, Modifier.fillMaxSize())
                        }
                        Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.xs)) {
                            Text(current.title, fontSize = Tokens.FontSize.title3, fontWeight = FontWeight.Bold, color = Tokens.palette.fg)
                            Row(horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm), verticalAlignment = Alignment.CenterVertically) {
                                Pill(stringResource(current.kind.title()))
                                scopeText(current.season, current.episode)?.let { Text(it, fontSize = Tokens.FontSize.caption, color = Tokens.palette.mutedFg) }
                            }
                            Text(
                                stringResource(if (current.isOpen) R.string.issue_open else R.string.issue_resolved), fontSize = Tokens.FontSize.subhead, fontWeight = FontWeight.SemiBold,
                                color = if (current.isOpen) Tokens.Status.requested else Tokens.Status.available,
                            )
                            if (!current.isOpen) current.resolvedBy?.let { Text(stringResource(R.string.issue_resolved_by, it.name), fontSize = Tokens.FontSize.footnote, color = Tokens.palette.mutedFg) }
                        }
                        Icon(Icons.AutoMirrored.Outlined.KeyboardArrowRight, null, tint = Tokens.palette.mutedFg)
                    }
                    Column(verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.md)) {
                        Text(stringResource(R.string.issue_thread), fontSize = Tokens.FontSize.title3, fontWeight = FontWeight.Bold, color = Tokens.palette.fg, modifier = Modifier.semantics { heading() })
                        if (current.comments.isEmpty()) Text(stringResource(R.string.m_admin_no_comments), fontSize = Tokens.FontSize.subhead, color = Tokens.palette.mutedFg)
                        current.comments.forEach { comment ->
                            Row(horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.md), verticalAlignment = Alignment.Top) {
                                AvatarView(comment.user.id, comment.user.name, 36.dp)
                                Column(verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.xs)) {
                                    Row(horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm), verticalAlignment = Alignment.CenterVertically) {
                                        Text(comment.user.name, fontSize = Tokens.FontSize.subhead, fontWeight = FontWeight.SemiBold, color = Tokens.palette.fg)
                                        Text(relativeTime(context, comment.createdAt), fontSize = Tokens.FontSize.caption, color = Tokens.palette.mutedFg)
                                    }
                                    Text(comment.message, fontSize = Tokens.FontSize.callout, color = Tokens.palette.fg)
                                }
                            }
                        }
                    }
                    Column(verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm)) {
                        OutlinedTextField(
                            value = reply, onValueChange = { reply = it }, modifier = Modifier.fillMaxWidth(),
                            placeholder = { Text(stringResource(R.string.issue_reply_placeholder), color = Tokens.palette.mutedFg) },
                            minLines = 2, maxLines = 6, shape = RoundedCornerShape(Tokens.Radius.md),
                            keyboardOptions = KeyboardOptions(capitalization = KeyboardCapitalization.Sentences),
                            colors = OutlinedTextFieldDefaults.colors(
                                focusedContainerColor = Tokens.palette.muted, unfocusedContainerColor = Tokens.palette.muted,
                                focusedBorderColor = Tokens.palette.ring, unfocusedBorderColor = Tokens.palette.border,
                                focusedTextColor = Tokens.palette.fg, unfocusedTextColor = Tokens.palette.fg, cursorColor = Tokens.palette.fg,
                            ),
                        )
                        TipsarrButton(
                            stringResource(if (sending) R.string.common_saving else R.string.issue_comment),
                            { sending = true; run { try { thread = api.comment(id, reply.trim()); reply = "" } finally { sending = false } } },
                            fullWidth = true, enabled = !sending && reply.isNotBlank(),
                        ) { Icon(Icons.AutoMirrored.Outlined.Send, null, Modifier.size(20.dp)); Spacer(Modifier.width(Tokens.Spacing.sm)) }
                    }
                    if (isAdmin) Column(verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm)) {
                        TipsarrButton(
                            stringResource(if (current.isOpen) R.string.issue_resolve else R.string.issue_reopen),
                            { run { thread = if (current.isOpen) api.resolveIssue(id) else api.reopenIssue(id) } },
                            fullWidth = true, kind = ButtonKind.Secondary,
                        ) { Icon(if (current.isOpen) Icons.Outlined.CheckCircle else Icons.Outlined.Undo, null, Modifier.size(20.dp)); Spacer(Modifier.width(Tokens.Spacing.sm)) }
                        TipsarrButton(stringResource(R.string.common_delete), { confirmDelete = true }, fullWidth = true, kind = ButtonKind.Ghost)
                    }
                }
            }
            failure != null -> if (failure == ApiError.Unreachable) OfflineState({ scope.launch { load() } })
            else StateView(Icons.Outlined.WarningAmber, failure.message(context), actionTitle = stringResource(R.string.common_retry), onAction = { scope.launch { load() } })
            else -> Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) { CircularProgressIndicator() }
        }
    }
    if (confirmDelete) {
        AlertDialog(
            onDismissRequest = { confirmDelete = false },
            title = { Text(stringResource(R.string.issue_confirm_delete)) },
            confirmButton = { TextButton({ confirmDelete = false; run { api.deleteIssue(id); onBack() } }) { Text(stringResource(R.string.common_delete), color = Tokens.palette.destructive) } },
            dismissButton = { TextButton({ confirmDelete = false }) { Text(stringResource(R.string.common_cancel)) } },
            containerColor = Tokens.palette.card,
        )
    }
}
