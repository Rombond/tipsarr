package com.brebond.tipsarr.features.admin

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.PlayArrow
import androidx.compose.material.icons.outlined.WarningAmber
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.SegmentedButton
import androidx.compose.material3.SegmentedButtonDefaults
import androidx.compose.material3.SingleChoiceSegmentedButtonRow
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
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.heading
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.brebond.tipsarr.R
import com.brebond.tipsarr.core.api.ApiError
import com.brebond.tipsarr.core.api.JobStatus
import com.brebond.tipsarr.core.api.Profile
import com.brebond.tipsarr.core.api.SyncStatus
import com.brebond.tipsarr.core.api.TipsarrApi
import com.brebond.tipsarr.core.api.UserDetail
import com.brebond.tipsarr.core.api.runJob
import com.brebond.tipsarr.core.api.setRole
import com.brebond.tipsarr.core.api.syncStatus
import com.brebond.tipsarr.core.api.userDetail
import com.brebond.tipsarr.core.api.users
import com.brebond.tipsarr.core.live.LocalLive
import com.brebond.tipsarr.core.live.OnTick
import com.brebond.tipsarr.core.support.LocalToast
import com.brebond.tipsarr.core.support.ToastKind
import com.brebond.tipsarr.core.support.relativeTime
import com.brebond.tipsarr.core.support.stringByName
import com.brebond.tipsarr.design.Tokens
import com.brebond.tipsarr.design.palette
import com.brebond.tipsarr.features.profile.AvatarView
import com.brebond.tipsarr.ui.Banner
import com.brebond.tipsarr.ui.BannerKind
import com.brebond.tipsarr.ui.ButtonKind
import com.brebond.tipsarr.ui.OfflineState
import com.brebond.tipsarr.ui.StateView
import com.brebond.tipsarr.ui.TipsarrButton
import com.brebond.tipsarr.ui.TopBarScaffold
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import java.text.SimpleDateFormat
import java.util.Date
import java.util.Locale

// Users

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun UsersScreen(api: TipsarrApi, onBack: () -> Unit, onOpen: (String) -> Unit) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    var users by remember { mutableStateOf<List<Profile>>(emptyList()) }
    var error by remember { mutableStateOf<ApiError?>(null) }
    var loaded by remember { mutableStateOf(false) }
    var refreshing by remember { mutableStateOf(false) }
    suspend fun load() {
        try { users = api.users().sortedByDescending { it.lastLoginAt }; error = null } catch (e: ApiError) { if (users.isEmpty()) error = e }
        loaded = true
    }
    LaunchedEffect(Unit) { load() }
    TopBarScaffold(stringResource(R.string.users_title), onBack) {
        PullToRefreshBox(isRefreshing = refreshing, onRefresh = { scope.launch { refreshing = true; load(); refreshing = false } }, modifier = Modifier.fillMaxSize()) {
            val failure = error
            when {
                failure != null -> if (failure == ApiError.Unreachable) OfflineState({ scope.launch { load() } })
                else StateView(Icons.Outlined.WarningAmber, failure.message(context), actionTitle = stringResource(R.string.common_retry), onAction = { scope.launch { load() } })
                !loaded -> Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) { CircularProgressIndicator() }
                else -> Box(Modifier.fillMaxSize().verticalScroll(rememberScrollState()).navigationBarsPadding(), contentAlignment = Alignment.TopCenter) {
                    Column(Modifier.widthIn(max = 720.dp).fillMaxWidth().padding(Tokens.Spacing.lg), verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.lg)) {
                        Text(stringResource(R.string.users_intro), fontSize = Tokens.FontSize.footnote, color = Tokens.palette.mutedFg)
                        Column(Modifier.fillMaxWidth().clip(RoundedCornerShape(Tokens.Radius.md)).background(Tokens.palette.card)) {
                            users.forEachIndexed { index, user ->
                                Row(
                                    Modifier.fillMaxWidth().clickable { onOpen(user.id) }.padding(Tokens.Spacing.md),
                                    horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.md), verticalAlignment = Alignment.CenterVertically,
                                ) {
                                    AvatarView(user.id, user.name, 40.dp)
                                    Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
                                        Row(horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm), verticalAlignment = Alignment.CenterVertically) {
                                            Text(user.name, fontSize = Tokens.FontSize.body, fontWeight = FontWeight.Medium, color = Tokens.palette.fg)
                                            if (user.isAdmin) Text(stringResource(R.string.users_role_admin), fontSize = 11.sp, fontWeight = FontWeight.SemiBold, color = Tokens.palette.fg,
                                                modifier = Modifier.clip(CircleShape).background(Tokens.palette.muted).padding(horizontal = Tokens.Spacing.sm, vertical = 2.dp))
                                        }
                                        if (user.lastLoginAt > 0) Text(stringResource(R.string.users_last_signin, relativeTime(context, user.lastLoginAt)), fontSize = Tokens.FontSize.footnote, color = Tokens.palette.mutedFg)
                                    }
                                }
                                if (index < users.lastIndex) HorizontalDivider(color = Tokens.palette.border)
                            }
                        }
                    }
                }
            }
        }
    }
}

@OptIn(ExperimentalLayoutApi::class, ExperimentalMaterial3Api::class)
@Composable
fun UserDetailScreen(api: TipsarrApi, id: String, myId: String, onBack: () -> Unit) {
    val context = LocalContext.current
    val toast = LocalToast.current
    val scope = rememberCoroutineScope()
    var detail by remember { mutableStateOf<UserDetail?>(null) }
    var error by remember { mutableStateOf<ApiError?>(null) }
    suspend fun load() { try { detail = api.userDetail(id); error = null } catch (e: ApiError) { if (detail == null) error = e } }
    LaunchedEffect(id) { load() }
    val isMe = id == myId

    TopBarScaffold(detail?.name.orEmpty(), onBack) {
        val current = detail
        val failure = error
        when {
            current != null -> Box(Modifier.fillMaxSize().verticalScroll(rememberScrollState()).navigationBarsPadding(), contentAlignment = Alignment.TopCenter) {
                Column(Modifier.widthIn(max = 720.dp).fillMaxWidth().padding(Tokens.Spacing.lg), verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.x2xl), horizontalAlignment = Alignment.CenterHorizontally) {
                    Column(horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm)) {
                        AvatarView(current.id, current.name, 88.dp)
                        Text(current.name, fontSize = Tokens.FontSize.title1, fontWeight = FontWeight.Bold, color = Tokens.palette.fg)
                        if (current.createdAt > 0) Text(
                            stringResource(R.string.profile_member_since, SimpleDateFormat("MMM yyyy", Locale.getDefault()).format(Date(current.createdAt * 1000))),
                            fontSize = Tokens.FontSize.footnote, color = Tokens.palette.mutedFg,
                        )
                    }
                    Column(Modifier.fillMaxWidth(), verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm)) {
                        Text(stringResource(R.string.users_role), fontSize = Tokens.FontSize.footnote, fontWeight = FontWeight.SemiBold, color = Tokens.palette.mutedFg)
                        val options = listOf(false to R.string.users_role_user, true to R.string.users_role_admin)
                        SingleChoiceSegmentedButtonRow(Modifier.fillMaxWidth()) {
                            options.forEachIndexed { index, (admin, label) ->
                                SegmentedButton(
                                    selected = current.isAdmin == admin, enabled = !isMe,
                                    onClick = {
                                        scope.launch {
                                            try { detail = current.copy(role = api.setRole(id, admin).role); toast?.show(context.getString(R.string.m_users_saved)) } catch (e: ApiError) { toast?.show(e.message(context), ToastKind.Error) }
                                        }
                                    },
                                    shape = SegmentedButtonDefaults.itemShape(index, options.size),
                                ) { Text(stringResource(label)) }
                            }
                        }
                        if (isMe) Text(stringResource(R.string.m_users_own_role), fontSize = Tokens.FontSize.footnote, color = Tokens.palette.mutedFg)
                    }
                    Column(Modifier.fillMaxWidth(), verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.md)) {
                        Text(stringResource(R.string.m_users_stats), fontSize = Tokens.FontSize.title3, fontWeight = FontWeight.Bold, color = Tokens.palette.fg, modifier = Modifier.semantics { heading() })
                        val s = current.stats
                        FlowRow(horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.md), verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.md), maxItemsInEachRow = 3) {
                            listOf(
                                R.string.m_users_requests to s.requests, R.string.m_users_pending to s.pending, R.string.m_users_approved to s.approved,
                                R.string.m_users_available to s.available, R.string.m_users_declined to s.declined, R.string.m_users_failed to s.failed,
                                R.string.m_users_movies to s.movies, R.string.m_users_shows to s.shows, R.string.m_users_watchlist to s.watchlist, R.string.m_users_watched to s.watched,
                            ).forEach { (label, value) -> Tile(stringResource(label), value, Modifier.weight(1f)) }
                        }
                    }
                }
            }
            failure != null -> if (failure == ApiError.Unreachable) OfflineState({ scope.launch { load() } })
            else StateView(Icons.Outlined.WarningAmber, failure.message(context), actionTitle = stringResource(R.string.common_retry), onAction = { scope.launch { load() } })
            else -> Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) { CircularProgressIndicator() }
        }
    }
}

@Composable
private fun Tile(label: String, value: Int, modifier: Modifier = Modifier) {
    val shape = RoundedCornerShape(Tokens.Radius.md)
    Column(
        modifier.widthIn(min = 96.dp).clip(shape).background(Tokens.palette.card).border(1.dp, Tokens.palette.border, shape).padding(vertical = Tokens.Spacing.md),
        horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.xs),
    ) {
        Text(value.toString(), fontSize = Tokens.FontSize.title2, fontWeight = FontWeight.Bold, color = Tokens.palette.fg)
        Text(label, fontSize = Tokens.FontSize.caption, color = Tokens.palette.mutedFg, textAlign = TextAlign.Center)
    }
}

// Sync

/** Background jobs of the server (library sync, history sync, box office, Radarr/Sonarr import, playback). */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SyncScreen(api: TipsarrApi, onBack: () -> Unit) {
    val context = LocalContext.current
    val toast = LocalToast.current
    val scope = rememberCoroutineScope()
    var status by remember { mutableStateOf<SyncStatus?>(null) }
    var error by remember { mutableStateOf<ApiError?>(null) }
    var refreshing by remember { mutableStateOf(false) }
    suspend fun load() { try { status = api.syncStatus(); error = null } catch (e: ApiError) { if (status == null) error = e } }
    LaunchedEffect(Unit) { load() }
    val live = LocalLive.current
    OnTick(live?.syncTick ?: 0) { load() }
    // Progress is pushed by the server; this only covers a dropped stream while a job runs.
    val anyRunning = status?.jobs?.any { it.running } == true
    LaunchedEffect(anyRunning) { while (anyRunning) { delay(5_000); load() } }

    TopBarScaffold(stringResource(R.string.m_admin_sync), onBack) {
        PullToRefreshBox(isRefreshing = refreshing, onRefresh = { scope.launch { refreshing = true; load(); refreshing = false } }, modifier = Modifier.fillMaxSize()) {
            val current = status
            val failure = error
            when {
                current != null -> Box(Modifier.fillMaxSize().verticalScroll(rememberScrollState()).navigationBarsPadding(), contentAlignment = Alignment.TopCenter) {
                    Column(Modifier.widthIn(max = 720.dp).fillMaxWidth().padding(Tokens.Spacing.lg), verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.lg)) {
                        Text(stringResource(R.string.m_sync_counts, current.movies.toString(), current.shows.toString()), fontSize = Tokens.FontSize.subhead, color = Tokens.palette.mutedFg)
                        if (!current.canSync) Banner(stringResource(R.string.m_sync_need_key), kind = BannerKind.Warning)
                        current.jobs.forEach { job ->
                            JobCard(job, canRun = current.canSync || job.name !in listOf("library-sync", "history-sync")) {
                                try { api.runJob(job.name); toast?.show(context.getString(R.string.m_sync_started)); load() } catch (e: ApiError) { toast?.show(e.message(context), ToastKind.Error) }
                            }
                        }
                    }
                }
                failure != null -> if (failure == ApiError.Unreachable) OfflineState({ scope.launch { load() } })
                else StateView(Icons.Outlined.WarningAmber, failure.message(context), actionTitle = stringResource(R.string.common_retry), onAction = { scope.launch { load() } })
                else -> Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) { CircularProgressIndicator() }
            }
        }
    }
}

@Composable
private fun JobCard(job: JobStatus, canRun: Boolean, run: suspend () -> Unit) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    var starting by remember { mutableStateOf(false) }
    val key = "m_job_" + job.name.replace('-', '_')
    val shape = RoundedCornerShape(Tokens.Radius.md)
    Column(
        Modifier.fillMaxWidth().clip(shape).background(Tokens.palette.card).border(1.dp, Tokens.palette.border, shape).padding(Tokens.Spacing.md),
        verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm),
    ) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Text(context.stringByName(key) ?: job.name, fontSize = Tokens.FontSize.headline, fontWeight = FontWeight.SemiBold, color = Tokens.palette.fg, modifier = Modifier.weight(1f))
            if (job.running) Row(horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.xs), verticalAlignment = Alignment.CenterVertically) {
                CircularProgressIndicator(Modifier.size(14.dp), strokeWidth = 2.dp, color = Tokens.Status.downloading)
                Text(stringResource(R.string.m_sync_running), fontSize = Tokens.FontSize.footnote, color = Tokens.Status.downloading)
            }
        }
        context.stringByName(key + "_desc")?.let { Text(it, fontSize = Tokens.FontSize.footnote, color = Tokens.palette.mutedFg) }
        Text(
            if (job.lastFinishedAt > 0) stringResource(R.string.m_sync_last, relativeTime(context, job.lastFinishedAt)) else stringResource(R.string.m_sync_never),
            fontSize = Tokens.FontSize.footnote, color = Tokens.palette.mutedFg,
        )
        if (job.message.isNotEmpty()) Text(job.message, fontSize = Tokens.FontSize.footnote, color = if (job.status == "error") Tokens.palette.destructive else Tokens.palette.mutedFg)
        Row(verticalAlignment = Alignment.CenterVertically) {
            if (job.everySeconds > 0) {
                val minutes = job.everySeconds / 60
                val every = if (minutes >= 60) "${minutes / 60} h" + (if (minutes % 60 > 0) " ${minutes % 60} min" else "") else "$minutes min"
                Text(stringResource(R.string.m_sync_every, every), fontSize = Tokens.FontSize.caption, color = Tokens.palette.mutedFg, modifier = Modifier.weight(1f))
            } else Box(Modifier.weight(1f))
            TipsarrButton(stringResource(R.string.m_sync_run), { starting = true; scope.launch { run(); starting = false } }, compact = true, kind = ButtonKind.Secondary, enabled = !job.running && !starting && canRun) {
                Icon(Icons.Outlined.PlayArrow, null, Modifier.size(16.dp))
            }
        }
    }
}
