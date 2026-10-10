package com.brebond.tipsarr.features.profile

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.Bookmark
import androidx.compose.material.icons.outlined.Checklist
import androidx.compose.material.icons.outlined.PhotoCamera
import androidx.compose.material.icons.outlined.RemoveRedEye
import androidx.compose.material.icons.outlined.Settings
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
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
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.heading
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.brebond.tipsarr.R
import com.brebond.tipsarr.core.api.MediaType
import com.brebond.tipsarr.core.api.Profile
import com.brebond.tipsarr.core.api.RequestRecord
import com.brebond.tipsarr.core.images.RemoteImage
import com.brebond.tipsarr.core.images.TmdbSize
import com.brebond.tipsarr.core.live.LocalLive
import com.brebond.tipsarr.core.live.OnTick
import com.brebond.tipsarr.core.support.plural
import com.brebond.tipsarr.core.support.relativeTime
import com.brebond.tipsarr.design.Tokens
import com.brebond.tipsarr.design.palette
import com.brebond.tipsarr.features.detail.MediaRoute
import com.brebond.tipsarr.ui.StatusBadge
import kotlinx.coroutines.launch
import java.text.DateFormat
import java.util.Date
import java.util.Locale

private val GUTTER = Tokens.Spacing.lg

/** Profile tab: who you are, three numbers, what you watched most, and your latest requests. */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ProfileScreen(
    model: ProfileModel,
    profile: Profile,
    avatarVersion: Int,
    onOpenSettings: () -> Unit,
    onOpenRequests: () -> Unit,
    onOpenWatchlist: () -> Unit,
    onOpenTitle: (MediaRoute) -> Unit,
    onOpenRequest: (RequestRecord) -> Unit,
    onChangePicture: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val scope = rememberCoroutineScope()
    var refreshing by remember { mutableStateOf(false) }
    LaunchedEffect(model) { model.load() }
    val live = LocalLive.current
    OnTick(live?.requestsTick ?: 0) { model.load() }
    PullToRefreshBox(
        isRefreshing = refreshing,
        onRefresh = { scope.launch { refreshing = true; model.load(); refreshing = false } },
        modifier = modifier.fillMaxSize(),
    ) {
        LazyColumn(Modifier.fillMaxSize(), contentPadding = PaddingValues(bottom = Tokens.Spacing.lg), verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.x2xl)) {
            item {
                Row(Modifier.fillMaxWidth().padding(start = GUTTER, end = Tokens.Spacing.sm, top = Tokens.Spacing.sm), verticalAlignment = Alignment.CenterVertically) {
                    Text(
                        stringResource(R.string.m_tab_profile), fontSize = Tokens.FontSize.largeTitle, fontWeight = FontWeight.Bold,
                        color = Tokens.palette.fg, modifier = Modifier.weight(1f).semantics { heading() },
                    )
                    IconButton(onOpenSettings) { Icon(Icons.Outlined.Settings, stringResource(R.string.nav_settings), tint = Tokens.palette.fg) }
                }
            }
            item { Header(profile, avatarVersion, onChangePicture) }
            item {
                Row(Modifier.padding(horizontal = GUTTER), horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.md)) {
                    StatTile(Icons.Outlined.Checklist, model.requestCount, stringResource(R.string.profile_stat_requests), Modifier.weight(1f), onOpenRequests)
                    StatTile(Icons.Outlined.Bookmark, model.watchlistCount, stringResource(R.string.profile_stat_watchlist), Modifier.weight(1f), onOpenWatchlist)
                    StatTile(Icons.Outlined.RemoveRedEye, model.watchedCount, stringResource(R.string.profile_stat_watched), Modifier.weight(1f))
                }
            }
            if (model.topWatched.isNotEmpty()) item {
                Column(verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.md)) {
                    Text(
                        stringResource(R.string.stats_top), fontSize = Tokens.FontSize.title3, fontWeight = FontWeight.Bold, color = Tokens.palette.fg,
                        modifier = Modifier.padding(horizontal = GUTTER).semantics { heading() },
                    )
                    LazyRow(horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.md), contentPadding = PaddingValues(horizontal = GUTTER)) {
                        items(model.topWatched.take(10), key = { "${it.type}-${it.tmdbId}" }) { top ->
                            TopWatched(top) { onOpenTitle(MediaRoute(top.type, top.tmdbId, top.title)) }
                        }
                    }
                }
            }
            if (model.recent.isNotEmpty()) item {
                Column(Modifier.padding(horizontal = GUTTER), verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.md)) {
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Text(
                            stringResource(R.string.profile_recent_requests), fontSize = Tokens.FontSize.title3, fontWeight = FontWeight.Bold,
                            color = Tokens.palette.fg, modifier = Modifier.weight(1f).semantics { heading() },
                        )
                        Text(stringResource(R.string.requests_tab_all), fontSize = Tokens.FontSize.subhead, color = Tokens.palette.mutedFg, modifier = Modifier.clickable(onClick = onOpenRequests).padding(Tokens.Spacing.sm))
                    }
                    Column {
                        model.recent.take(3).forEach { record ->
                            Row(
                                Modifier.fillMaxWidth().clickable { onOpenRequest(record) }.padding(vertical = Tokens.Spacing.sm),
                                horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.md), verticalAlignment = Alignment.CenterVertically,
                            ) {
                                Box(Modifier.width(44.dp).height(66.dp).clip(RoundedCornerShape(Tokens.Radius.sm - 2.dp)).background(Tokens.palette.muted)) {
                                    RemoteImage(record.posterPath, TmdbSize.W92, Modifier.fillMaxSize())
                                }
                                Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.xs)) {
                                    Text(record.title, fontSize = Tokens.FontSize.body, fontWeight = FontWeight.SemiBold, color = Tokens.palette.fg, maxLines = 1, overflow = TextOverflow.Ellipsis)
                                    StatusBadge(record.state)
                                }
                            }
                            HorizontalDivider(color = Tokens.palette.border)
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun Header(profile: Profile, avatarVersion: Int, onChangePicture: () -> Unit) {
    val context = LocalContext.current
    val member = buildList {
        if (profile.createdAt > 0) {
            val month = java.text.SimpleDateFormat("MMM yyyy", Locale.getDefault()).format(Date(profile.createdAt * 1000))
            add(stringResource(R.string.profile_member_since, month))
        }
        if (profile.lastLoginAt > 0) add(stringResource(R.string.profile_last_seen, relativeTime(context, profile.lastLoginAt)))
    }.joinToString(" · ")
    Column(Modifier.fillMaxWidth(), horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.md)) {
        Box(Modifier.clickable(onClick = onChangePicture).semantics { }) {
            AvatarView(profile.id, profile.name, 88.dp, avatarVersion)
            Icon(
                Icons.Outlined.PhotoCamera, stringResource(R.string.profile_change_picture), tint = Tokens.palette.primaryFg,
                modifier = Modifier.align(Alignment.BottomEnd).size(28.dp).clip(CircleShape).background(Tokens.palette.primary).padding(6.dp),
            )
        }
        Text(profile.name, fontSize = Tokens.FontSize.title1, fontWeight = FontWeight.Bold, color = Tokens.palette.fg)
        if (member.isNotEmpty()) Text(member, fontSize = Tokens.FontSize.footnote, color = Tokens.palette.mutedFg, textAlign = TextAlign.Center, modifier = Modifier.padding(horizontal = GUTTER))
    }
}

@Composable
private fun StatTile(icon: ImageVector, value: Int?, label: String, modifier: Modifier = Modifier, onClick: (() -> Unit)? = null) {
    val shape = RoundedCornerShape(Tokens.Radius.md)
    Column(
        modifier.clip(shape).background(Tokens.palette.card).border(1.dp, Tokens.palette.border, shape)
            .then(if (onClick != null) Modifier.clickable(onClick = onClick) else Modifier).padding(vertical = Tokens.Spacing.md),
        horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.xs),
    ) {
        Icon(icon, null, tint = Tokens.palette.mutedFg)
        Text(value?.toString() ?: "–", fontSize = Tokens.FontSize.title2, fontWeight = FontWeight.Bold, color = Tokens.palette.fg)
        Text(label, fontSize = Tokens.FontSize.footnote, color = Tokens.palette.mutedFg)
    }
}

@Composable
private fun TopWatched(top: com.brebond.tipsarr.core.api.StatsTop, onClick: () -> Unit) {
    val context = LocalContext.current
    val hours = String.format(Locale.getDefault(), "%.0f", top.hours) + " " + stringResource(R.string.stats_unit_h)
    val plays = context.getString(if (top.type == MediaType.Tv) R.string.stats_n_episodes else R.string.stats_n_plays, top.plays.toString())
    Column(Modifier.width(120.dp).clickable(onClick = onClick), verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm)) {
        Box(Modifier.fillMaxWidth().height(180.dp).clip(RoundedCornerShape(Tokens.Radius.md)).background(Tokens.palette.muted)) {
            RemoteImage(top.posterPath, TmdbSize.W342, Modifier.fillMaxSize(), server = true)
        }
        Text(top.title, fontSize = Tokens.FontSize.subhead, fontWeight = FontWeight.Medium, color = Tokens.palette.fg, maxLines = 1, overflow = TextOverflow.Ellipsis)
        Text("$hours · $plays", fontSize = Tokens.FontSize.footnote, color = Tokens.palette.mutedFg, maxLines = 1, overflow = TextOverflow.Ellipsis)
    }
}
