package com.brebond.tipsarr.features.profile

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.outlined.KeyboardArrowRight
import androidx.compose.material.icons.outlined.BarChart
import androidx.compose.material.icons.outlined.Bookmark
import androidx.compose.material.icons.outlined.ChatBubbleOutline
import androidx.compose.material.icons.outlined.Checklist
import androidx.compose.material.icons.outlined.Dns
import androidx.compose.material.icons.outlined.History
import androidx.compose.material.icons.outlined.People
import androidx.compose.material.icons.outlined.RemoveRedEye
import androidx.compose.material.icons.outlined.Settings
import androidx.compose.material.icons.outlined.Sync
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.heading
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import com.brebond.tipsarr.R
import com.brebond.tipsarr.core.api.Permission
import com.brebond.tipsarr.core.api.Profile
import com.brebond.tipsarr.core.api.RequestRecord
import com.brebond.tipsarr.core.images.RemoteImage
import com.brebond.tipsarr.core.images.TmdbSize
import com.brebond.tipsarr.core.support.relativeTime
import com.brebond.tipsarr.design.Tokens
import com.brebond.tipsarr.design.palette
import com.brebond.tipsarr.features.detail.MediaRoute
import com.brebond.tipsarr.ui.StatusBadge

private val GUTTER = Tokens.Spacing.lg

/**
 * Profile on a wide screen: the two tablet layouts of Penpot page 10 (the iOS `ProfileRegular`).
 * Portrait ([wide] false): one wide column. Landscape: the profile card, numbers and Settings on the left, the dashboard on the right.
 */
@Composable
internal fun ProfileWide(
    model: ProfileModel,
    profile: Profile,
    avatarVersion: Int,
    wide: Boolean,
    server: String?,
    serverVersion: String?,
    onOpenRequests: () -> Unit,
    onOpenWatchlist: () -> Unit,
    onOpenTitle: (MediaRoute) -> Unit,
    onOpenRequest: (RequestRecord) -> Unit,
    onOpenStats: () -> Unit,
    onOpenIssues: () -> Unit,
    onOpenUsers: () -> Unit,
    onOpenSync: () -> Unit,
    teasersOn: Boolean,
    onOpenTeasersHistory: () -> Unit,
    onOpenSettings: (() -> Unit)?,
    onBack: (() -> Unit)?,
) {
    val dashboard: @Composable (Dp) -> Unit = { posterWidth ->
        MostWatched(model, posterWidth, onOpenStats, onOpenTitle)
        RecentGrid(model.recent, onOpenRequests, onOpenRequest)
        if (teasersOn) RowGroup(stringResource(R.string.m_tab_teasers), listOf { AdminRow(Icons.Outlined.History, Tokens.Status.approved, stringResource(R.string.m_teasers_history_title), null, onOpenTeasersHistory) }, gutter = 0.dp)
        AdminTiles(profile, model.openIssues, onOpenIssues, onOpenUsers, onOpenSync, onOpenStats)
    }
    Box(Modifier.fillMaxSize().verticalScroll(rememberScrollState()), contentAlignment = Alignment.TopCenter) {
        Column(Modifier.widthIn(max = 1100.dp).fillMaxWidth().padding(bottom = Tokens.Spacing.x2xl), verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.lg)) {
            TitleRow(onBack)
            if (wide) {
                Row(Modifier.padding(horizontal = GUTTER), horizontalArrangement = Arrangement.spacedBy(28.dp), verticalAlignment = Alignment.Top) {
                    Column(Modifier.width(300.dp), verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.lg)) {
                        HeaderCard(profile, avatarVersion, server, serverVersion, vertical = true)
                        StatTiles(model, vertical = true, onOpenRequests, onOpenWatchlist)
                        if (onOpenSettings != null) SettingsCard(onOpenSettings)
                    }
                    Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.x2xl)) { dashboard(114.dp) }
                }
            } else {
                Column(Modifier.padding(horizontal = GUTTER), verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.x2xl)) {
                    Column(verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.lg)) {
                        HeaderCard(profile, avatarVersion, server, serverVersion, vertical = false)
                        StatTiles(model, vertical = false, onOpenRequests, onOpenWatchlist)
                    }
                    dashboard(116.dp)
                    if (onOpenSettings != null) SettingsCard(onOpenSettings)
                }
            }
        }
    }
}

@Composable
private fun Modifier.card(radius: Dp = Tokens.Radius.md): Modifier {
    val shape = RoundedCornerShape(radius)
    return this.clip(shape).border(1.dp, Tokens.palette.border, shape)
}

/** Profile card: avatar, name, role, dates and the server. Vertical in the landscape column, horizontal in portrait. */
@Composable
private fun HeaderCard(profile: Profile, avatarVersion: Int, server: String?, serverVersion: String?, vertical: Boolean) {
    val role = stringResource(if (profile.isAdmin) R.string.nav_administrator else R.string.nav_member)
    val member = memberLine(profile)
    val serverBlock: @Composable () -> Unit = {
        if (server != null) Row(horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm), verticalAlignment = Alignment.CenterVertically) {
            Icon(Icons.Outlined.Dns, null, tint = Tokens.palette.mutedFg, modifier = Modifier.size(18.dp))
            Column {
                Text(server, fontSize = Tokens.FontSize.subhead, fontWeight = FontWeight.SemiBold, color = Tokens.palette.fg, maxLines = 1, overflow = TextOverflow.Ellipsis)
                serverVersion?.let { Text(stringResource(R.string.m_settings_server_version, it), fontSize = Tokens.FontSize.caption, color = Tokens.palette.mutedFg) }
            }
        }
    }
    val modifier = Modifier.fillMaxWidth().card(Tokens.Radius.xl).background(Tokens.palette.card).padding(Tokens.Spacing.xl)
    if (vertical) {
        Column(modifier, horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.md)) {
            AvatarView(profile.id, profile.name, 104.dp, avatarVersion)
            Text(profile.name, fontSize = Tokens.FontSize.title1, fontWeight = FontWeight.Bold, color = Tokens.palette.fg, textAlign = TextAlign.Center)
            Text(role, fontSize = Tokens.FontSize.subhead, color = Tokens.palette.mutedFg)
            if (member.isNotEmpty()) Text(member, fontSize = Tokens.FontSize.footnote, color = Tokens.palette.mutedFg, textAlign = TextAlign.Center)
            if (server != null) {
                HorizontalDivider(color = Tokens.palette.border)
                Box(Modifier.fillMaxWidth()) { serverBlock() }
            }
        }
    } else {
        Row(modifier, horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.xl), verticalAlignment = Alignment.CenterVertically) {
            AvatarView(profile.id, profile.name, 96.dp, avatarVersion)
            Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm)) {
                Text(profile.name, fontSize = Tokens.FontSize.title1, fontWeight = FontWeight.Bold, color = Tokens.palette.fg)
                Text(listOf(role, member).filter { it.isNotEmpty() }.joinToString(" · "), fontSize = Tokens.FontSize.subhead, color = Tokens.palette.mutedFg)
                serverBlock()
            }
        }
    }
}

/** Requests, watchlist, watched: stacked tiles in the landscape column, icon beside the number in portrait. */
@Composable
private fun StatTiles(model: ProfileModel, vertical: Boolean, onOpenRequests: () -> Unit, onOpenWatchlist: () -> Unit) {
    val tiles = listOf(
        Triple(Icons.Outlined.Checklist, model.requestCount, R.string.profile_stat_requests) to onOpenRequests,
        Triple(Icons.Outlined.Bookmark, model.watchlistCount, R.string.profile_stat_watchlist) to onOpenWatchlist,
        Triple(Icons.Outlined.RemoveRedEye, model.watchedCount, R.string.profile_stat_watched) to null,
    )
    Row(horizontalArrangement = Arrangement.spacedBy(if (vertical) Tokens.Spacing.md else Tokens.Spacing.lg)) {
        tiles.forEach { (tile, onClick) ->
            val (icon, value, label) = tile
            if (vertical) {
                StatTile(icon, value, stringResource(label), Modifier.weight(1f), onClick)
            } else {
                Row(
                    Modifier.weight(1f).card().background(Tokens.palette.card)
                        .then(if (onClick != null) Modifier.clickable(onClick = onClick) else Modifier).padding(Tokens.Spacing.lg),
                    horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.md), verticalAlignment = Alignment.CenterVertically,
                ) {
                    Icon(icon, null, tint = Tokens.palette.mutedFg, modifier = Modifier.size(28.dp))
                    Column {
                        Text(value?.toString() ?: "–", fontSize = Tokens.FontSize.title2, fontWeight = FontWeight.Bold, color = Tokens.palette.fg)
                        Text(stringResource(label), fontSize = Tokens.FontSize.footnote, color = Tokens.palette.mutedFg, maxLines = 1, overflow = TextOverflow.Ellipsis)
                    }
                }
            }
        }
    }
}

@Composable
private fun SettingsCard(onClick: () -> Unit) {
    Row(
        Modifier.fillMaxWidth().card().background(Tokens.palette.card).clickable(onClick = onClick)
            .padding(horizontal = Tokens.Spacing.md, vertical = Tokens.Spacing.md),
        horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.md), verticalAlignment = Alignment.CenterVertically,
    ) {
        Icon(Icons.Outlined.Settings, null, tint = Tokens.palette.fg)
        Text(stringResource(R.string.nav_settings), fontSize = Tokens.FontSize.body, color = Tokens.palette.fg, modifier = Modifier.weight(1f))
        Icon(Icons.AutoMirrored.Outlined.KeyboardArrowRight, null, tint = Tokens.palette.mutedFg)
    }
}

@Composable
private fun SectionHeader(title: String, action: String, onAction: () -> Unit) {
    Row(verticalAlignment = Alignment.CenterVertically) {
        Text(title, fontSize = Tokens.FontSize.title3, fontWeight = FontWeight.Bold, color = Tokens.palette.fg, modifier = Modifier.weight(1f).semantics { heading() })
        Text(action, fontSize = Tokens.FontSize.subhead, color = Tokens.palette.mutedFg, modifier = Modifier.clickable(onClick = onAction).padding(Tokens.Spacing.sm))
    }
}

@Composable
private fun MostWatched(model: ProfileModel, posterWidth: Dp, onOpenStats: () -> Unit, onOpenTitle: (MediaRoute) -> Unit) {
    if (model.topWatched.isEmpty()) return
    Column(verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.md)) {
        SectionHeader(stringResource(R.string.stats_top), stringResource(R.string.m_stats_open), onOpenStats)
        LazyRow(horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.md)) {
            items(model.topWatched.take(10), key = { "${it.type}-${it.tmdbId}" }) { top ->
                TopWatched(top, posterWidth) { onOpenTitle(MediaRoute(top.type, top.tmdbId, top.title)) }
            }
        }
    }
}

/** Recent requests as cards, two per row. */
@Composable
private fun RecentGrid(recent: List<RequestRecord>, onOpenRequests: () -> Unit, onOpenRequest: (RequestRecord) -> Unit) {
    if (recent.isEmpty()) return
    val context = LocalContext.current
    Column(verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.md)) {
        SectionHeader(stringResource(R.string.profile_recent_requests), stringResource(R.string.requests_tab_all), onOpenRequests)
        recent.take(4).chunked(2).forEach { pair ->
            Row(horizontalArrangement = Arrangement.spacedBy(14.dp)) {
                pair.forEach { record ->
                    Row(
                        Modifier.weight(1f).card().background(Tokens.palette.card).clickable { onOpenRequest(record) }.padding(Tokens.Spacing.md),
                        horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.md), verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Box(Modifier.width(44.dp).height(66.dp).clip(RoundedCornerShape(Tokens.Radius.sm - 2.dp)).background(Tokens.palette.muted)) {
                            RemoteImage(record.posterPath, TmdbSize.W92, Modifier.fillMaxSize())
                        }
                        Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.xs)) {
                            Text(record.title, fontSize = Tokens.FontSize.body, fontWeight = FontWeight.SemiBold, color = Tokens.palette.fg, maxLines = 1, overflow = TextOverflow.Ellipsis)
                            StatusBadge(record.state)
                            if (record.createdAt > 0) Text(relativeTime(context, record.createdAt), fontSize = Tokens.FontSize.caption, color = Tokens.palette.mutedFg, maxLines = 1)
                        }
                    }
                }
                if (pair.size == 1) Spacer(Modifier.weight(1f))
            }
        }
    }
}

/** The administrator tools as tiles in a row (wide screens); the phone has the list card instead. */
@Composable
private fun AdminTiles(profile: Profile, openIssues: Int?, onOpenIssues: () -> Unit, onOpenUsers: () -> Unit, onOpenSync: () -> Unit, onOpenStats: () -> Unit) {
    val tiles = buildList {
        if (profile.can(Permission.VIEW_ISSUES)) add(AdminTile(Icons.Outlined.ChatBubbleOutline, Tokens.Status.failed, R.string.nav_issues, openIssues, onOpenIssues))
        if (profile.can(Permission.MANAGE_USERS)) add(AdminTile(Icons.Outlined.People, Tokens.Status.approved, R.string.nav_users, null, onOpenUsers))
        if (profile.can(Permission.MANAGE_SETTINGS)) add(AdminTile(Icons.Outlined.Sync, Tokens.Status.searching, R.string.m_admin_sync, null, onOpenSync))
        if (profile.can(Permission.MANAGE_USERS)) add(AdminTile(Icons.Outlined.BarChart, Tokens.Status.downloading, R.string.nav_stats, null, onOpenStats))
    }
    if (tiles.isEmpty()) return
    Column(verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm)) {
        Text(stringResource(R.string.m_admin_title).uppercase(), fontSize = Tokens.FontSize.footnote, fontWeight = FontWeight.SemiBold, color = Tokens.palette.mutedFg)
        Row(horizontalArrangement = Arrangement.spacedBy(14.dp)) {
            tiles.forEach { tile ->
                Column(
                    Modifier.weight(1f).card().background(Tokens.palette.card).clickable(onClick = tile.onClick).padding(Tokens.Spacing.md),
                    verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.md),
                ) {
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Icon(
                            tile.icon, null, tint = tile.color,
                            modifier = Modifier.size(34.dp).clip(RoundedCornerShape(10.dp)).background(tile.color.copy(alpha = 0.16f)).padding(7.dp),
                        )
                        Spacer(Modifier.weight(1f))
                        if (tile.count != null && tile.count > 0) Text(tile.count.toString(), fontSize = Tokens.FontSize.title3, fontWeight = FontWeight.Bold, color = Tokens.palette.fg)
                    }
                    Text(stringResource(tile.title), fontSize = Tokens.FontSize.subhead, fontWeight = FontWeight.SemiBold, color = Tokens.palette.fg, maxLines = 2, overflow = TextOverflow.Ellipsis)
                }
            }
        }
    }
}

private class AdminTile(val icon: ImageVector, val color: Color, val title: Int, val count: Int?, val onClick: () -> Unit)
