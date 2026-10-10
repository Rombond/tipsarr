package com.brebond.tipsarr.features.admin

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.BarChart
import androidx.compose.material.icons.outlined.Info
import androidx.compose.material.icons.outlined.KeyboardArrowDown
import androidx.compose.material.icons.outlined.WarningAmber
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
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
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.brebond.tipsarr.R
import com.brebond.tipsarr.core.api.ApiError
import com.brebond.tipsarr.core.api.FullStats
import com.brebond.tipsarr.core.api.LibraryPage
import com.brebond.tipsarr.core.api.Profile
import com.brebond.tipsarr.core.api.RequestPage
import com.brebond.tipsarr.core.api.StatsBucket
import com.brebond.tipsarr.core.api.StatsMonth
import com.brebond.tipsarr.core.api.StatsPeriod
import com.brebond.tipsarr.core.api.TipsarrApi
import com.brebond.tipsarr.core.api.neverWatched
import com.brebond.tipsarr.core.api.stats
import com.brebond.tipsarr.core.api.unwatchedRequests
import com.brebond.tipsarr.core.api.users
import com.brebond.tipsarr.design.Tokens
import com.brebond.tipsarr.design.palette
import com.brebond.tipsarr.features.detail.MediaRoute
import com.brebond.tipsarr.ui.OfflineState
import com.brebond.tipsarr.ui.StateView
import com.brebond.tipsarr.ui.TopBarScaffold
import kotlinx.coroutines.launch
import java.text.DateFormatSymbols
import java.util.Locale

/**
 * Watching statistics, in the order of the web page: tiles, most watched (all, movies, shows), charts,
 * "asked for, never watched". Everyone sees their own; admins can pick someone or everyone.
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun StatsScreen(
    api: TipsarrApi,
    isAdmin: Boolean,
    me: Profile,
    onBack: () -> Unit,
    onOpenTitle: (MediaRoute) -> Unit,
    onSeeAll: (PosterListRoute) -> Unit,
) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    var period by remember { mutableStateOf(StatsPeriod.All) }
    /** null = me, "all" = everyone, else a user id. */
    var who by remember { mutableStateOf<String?>(null) }
    var users by remember { mutableStateOf<List<Profile>>(emptyList()) }
    var report by remember { mutableStateOf<FullStats?>(null) }
    var unwatched by remember { mutableStateOf<RequestPage?>(null) }
    var never by remember { mutableStateOf<LibraryPage?>(null) }
    var error by remember { mutableStateOf<ApiError?>(null) }
    var loading by remember { mutableStateOf(true) }
    var refreshing by remember { mutableStateOf(false) }

    suspend fun load() {
        loading = true
        try { report = api.stats(period, who); error = null } catch (e: ApiError) { if (report == null) error = e }
        loading = false
        // Who the "never watched" list is about: the person, or everyone when an admin picked "everyone".
        val target = if (who == "all") null else (who ?: me.id)
        unwatched = runCatching { api.unwatchedRequests(target) }.getOrNull()
        if (isAdmin && never == null) never = runCatching { api.neverWatched() }.getOrNull()
    }
    LaunchedEffect(period, who) { load() }
    LaunchedEffect(isAdmin) { if (isAdmin) users = runCatching { api.users() }.getOrDefault(emptyList()) }

    TopBarScaffold(stringResource(R.string.stats_title), onBack) {
        PullToRefreshBox(isRefreshing = refreshing, onRefresh = { scope.launch { refreshing = true; load(); refreshing = false } }, modifier = Modifier.fillMaxSize()) {
            Box(Modifier.fillMaxSize().verticalScroll(rememberScrollState()).navigationBarsPadding(), contentAlignment = Alignment.TopCenter) {
                Column(Modifier.widthIn(max = 720.dp).fillMaxWidth().padding(Tokens.Spacing.lg), verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.x2xl)) {
                    Controls(report, period, { period = it }, isAdmin, users.filter { it.id != me.id }, who, { who = it })
                    val current = report
                    val failure = error
                    when {
                        current != null -> Content(current, isAdmin, onOpenTitle, onSeeAll)
                        failure != null -> Box(Modifier.fillMaxWidth().height(300.dp)) {
                            if (failure == ApiError.Unreachable) OfflineState({ scope.launch { load() } })
                            else StateView(Icons.Outlined.WarningAmber, failure.message(context), actionTitle = stringResource(R.string.common_retry), onAction = { scope.launch { load() } })
                        }
                        loading -> Box(Modifier.fillMaxWidth().height(200.dp), contentAlignment = Alignment.Center) { CircularProgressIndicator() }
                    }
                    unwatched?.let { page ->
                        Column(verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.md)) {
                            if (page.items.isEmpty()) {
                                Heading(stringResource(R.string.stats_unwatched_title))
                                Text(stringResource(R.string.stats_unwatched_none), fontSize = Tokens.FontSize.subhead, color = Tokens.palette.mutedFg)
                            } else {
                                PosterCarousel(R.string.stats_unwatched_title, page.items.map { it.toUnwatchedRow(context) }, onOpenTitle, onSeeAll)
                                Text(stringResource(if (who == null || !isAdmin) R.string.stats_unwatched_mine else R.string.stats_unwatched_others), fontSize = Tokens.FontSize.footnote, color = Tokens.palette.mutedFg)
                                if (page.total > page.items.size) Text(stringResource(R.string.stats_unwatched_more, (page.total - page.items.size).toString()), fontSize = Tokens.FontSize.footnote, color = Tokens.palette.mutedFg)
                            }
                        }
                    }
                    if (isAdmin) never?.let { page ->
                        Column(verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.md)) {
                            if (page.items.isEmpty()) {
                                Heading(stringResource(R.string.stats_never_title))
                                Text(stringResource(R.string.stats_never_none), fontSize = Tokens.FontSize.subhead, color = Tokens.palette.mutedFg)
                            } else {
                                PosterCarousel(R.string.stats_never_title, page.items.map { it.toRow() }, onOpenTitle, onSeeAll)
                                Text(stringResource(R.string.stats_never_hint, page.total.toString()), fontSize = Tokens.FontSize.footnote, color = Tokens.palette.mutedFg)
                            }
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun Heading(text: String) {
    Text(text, fontSize = Tokens.FontSize.title3, fontWeight = FontWeight.Bold, color = Tokens.palette.fg, modifier = Modifier.semantics { heading() })
}

@Composable
private fun Controls(report: FullStats?, period: StatsPeriod, onPeriod: (StatsPeriod) -> Unit, isAdmin: Boolean, users: List<Profile>, who: String?, onWho: (String?) -> Unit) {
    Column(verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.md)) {
        if (report != null) Text(stringResource(if (report.exact) R.string.stats_source_plugin else R.string.stats_source_estimate), fontSize = Tokens.FontSize.footnote, color = Tokens.palette.mutedFg)
        val options = listOf(StatsPeriod.Days30 to R.string.stats_period_30d, StatsPeriod.Months12 to R.string.stats_period_12m, StatsPeriod.All to R.string.stats_period_all)
        SingleChoiceSegmentedButtonRow(Modifier.fillMaxWidth()) {
            options.forEachIndexed { index, (value, label) ->
                SegmentedButton(selected = period == value, onClick = { onPeriod(value) }, shape = SegmentedButtonDefaults.itemShape(index, options.size)) { Text(stringResource(label), maxLines = 1, overflow = TextOverflow.Ellipsis) }
            }
        }
        if (isAdmin && users.isNotEmpty()) {
            var open by remember { mutableStateOf(false) }
            val label = when (who) { null -> stringResource(R.string.stats_me); "all" -> stringResource(R.string.stats_everyone); else -> users.firstOrNull { it.id == who }?.name.orEmpty() }
            Box {
                Row(
                    Modifier.fillMaxWidth().clip(RoundedCornerShape(Tokens.Radius.md)).background(Tokens.palette.muted).clickable { open = true }.padding(Tokens.Spacing.md),
                    horizontalArrangement = Arrangement.SpaceBetween, verticalAlignment = Alignment.CenterVertically,
                ) {
                    Text(stringResource(R.string.stats_user), fontSize = Tokens.FontSize.body, color = Tokens.palette.mutedFg)
                    Text(label, fontSize = Tokens.FontSize.body, color = Tokens.palette.fg)
                    Icon(Icons.Outlined.KeyboardArrowDown, null, tint = Tokens.palette.mutedFg)
                }
                DropdownMenu(open, { open = false }, containerColor = Tokens.palette.card) {
                    DropdownMenuItem(text = { Text(stringResource(R.string.stats_me), color = Tokens.palette.fg) }, onClick = { open = false; onWho(null) })
                    DropdownMenuItem(text = { Text(stringResource(R.string.stats_everyone), color = Tokens.palette.fg) }, onClick = { open = false; onWho("all") })
                    users.forEach { user -> DropdownMenuItem(text = { Text(user.name, color = Tokens.palette.fg) }, onClick = { open = false; onWho(user.id) }) }
                }
            }
        }
    }
}

@Composable
private fun Content(r: FullStats, isAdmin: Boolean, onOpenTitle: (MediaRoute) -> Unit, onSeeAll: (PosterListRoute) -> Unit) {
    val context = LocalContext.current
    val unit = stringResource(R.string.stats_unit_h)
    if (isAdmin && r.plugin.hint) {
        Column(Modifier.fillMaxWidth().clip(RoundedCornerShape(Tokens.Radius.md)).background(Tokens.palette.muted).padding(Tokens.Spacing.md), verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm)) {
            Row(horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm), verticalAlignment = Alignment.CenterVertically) {
                Icon(Icons.Outlined.Info, null, tint = Tokens.palette.fg)
                Text(stringResource(R.string.stats_plugin_title), fontSize = Tokens.FontSize.headline, fontWeight = FontWeight.SemiBold, color = Tokens.palette.fg)
            }
            Text(stringResource(R.string.stats_plugin_text), fontSize = Tokens.FontSize.footnote, color = Tokens.palette.mutedFg)
            Text(stringResource(R.string.stats_plugin_step1), fontSize = Tokens.FontSize.footnote, color = Tokens.palette.fg)
            Text(stringResource(R.string.stats_plugin_step2), fontSize = Tokens.FontSize.footnote, color = Tokens.palette.fg)
            Text(stringResource(R.string.stats_plugin_step3), fontSize = Tokens.FontSize.footnote, color = Tokens.palette.fg)
        }
    }
    if (r.totals.plays == 0 && r.totals.titles == 0) {
        StateView(Icons.Outlined.BarChart, stringResource(R.string.stats_empty), Modifier.height(240.dp))
        return
    }
    Column(verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.md)) {
        Row(horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.md)) {
            StatTile(stringResource(if (r.exact) R.string.stats_tile_hours else R.string.stats_tile_hours_est), StatsFormat.hours(r.totals.hours, unit), stringResource(R.string.stats_unit_hours_long), Modifier.weight(1f))
            StatTile(stringResource(R.string.stats_tile_titles), r.totals.titles.toString(), stringResource(R.string.stats_tile_titles_sub, r.totals.movies.toString(), r.totals.shows.toString()), Modifier.weight(1f))
        }
        Row(horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.md)) {
            StatTile(stringResource(R.string.stats_tile_plays), r.totals.plays.toString(), r.genres.firstOrNull()?.let { stringResource(R.string.stats_tile_plays_sub, it.name) }, Modifier.weight(1f))
            StatTile(stringResource(R.string.stats_tile_requests), r.requests.made.toString(), stringResource(R.string.stats_tile_requests_sub, r.requests.available.toString(), r.requests.declined.toString()), Modifier.weight(1f))
        }
    }
    PosterCarousel(R.string.stats_top, r.top.map { it.toRow(context) }, onOpenTitle, onSeeAll, ranked = true)
    PosterCarousel(R.string.stats_top_movies, r.topMovies.map { it.toRow(context) }, onOpenTitle, onSeeAll, ranked = true)
    PosterCarousel(R.string.stats_top_shows, r.topShows.map { it.toRow(context) }, onOpenTitle, onSeeAll, ranked = true)
    if (r.genres.isNotEmpty()) ChartCard(stringResource(R.string.stats_genres)) { Buckets(r.genres.take(8), byHours = true, unit = unit) }
    if (r.decades.isNotEmpty()) ChartCard(stringResource(R.string.stats_decades)) { Buckets(r.decades.take(8), byHours = false, unit = unit) }
    if (r.months.isNotEmpty()) ChartCard(stringResource(R.string.stats_months)) { Columns(r.months.map { monthLabel(it) to it.hours }) }
    if (r.weekdays.any { it > 0 }) ChartCard(stringResource(R.string.stats_weekdays)) {
        // The server counts Monday first.
        val symbols = DateFormatSymbols.getInstance(Locale.getDefault()).shortWeekdays.drop(1)
        val names = symbols.drop(1) + symbols.take(1)
        Columns(r.weekdays.take(7).mapIndexed { i, v -> names.getOrElse(i) { "" } to v })
    }
    if (r.hoursOfDay.any { it > 0 }) ChartCard(stringResource(R.string.stats_hours_of_day)) {
        Columns(r.hoursOfDay.mapIndexed { i, v -> (if (i % 3 == 0) i.toString() else "") to v })
    }
}

private fun monthLabel(m: StatsMonth): String = runCatching {
    val (year, month) = m.month.split("-")
    DateFormatSymbols.getInstance(Locale.getDefault()).shortMonths[month.toInt() - 1]
}.getOrDefault(m.month)

@Composable
private fun StatTile(label: String, value: String, sub: String?, modifier: Modifier = Modifier) {
    val shape = RoundedCornerShape(Tokens.Radius.md)
    Column(modifier.clip(shape).background(Tokens.palette.card).border(1.dp, Tokens.palette.border, shape).padding(Tokens.Spacing.md), verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.xs)) {
        Text(label, fontSize = Tokens.FontSize.footnote, color = Tokens.palette.mutedFg)
        Text(value, fontSize = Tokens.FontSize.title1, fontWeight = FontWeight.Bold, color = Tokens.palette.fg)
        if (sub != null) Text(sub, fontSize = Tokens.FontSize.caption, color = Tokens.palette.mutedFg)
    }
}

@Composable
private fun ChartCard(title: String, chart: @Composable () -> Unit) {
    val shape = RoundedCornerShape(Tokens.Radius.md)
    Column(Modifier.fillMaxWidth().clip(shape).background(Tokens.palette.card).border(1.dp, Tokens.palette.border, shape).padding(Tokens.Spacing.md), verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.md)) {
        Text(title, fontSize = Tokens.FontSize.headline, fontWeight = FontWeight.SemiBold, color = Tokens.palette.fg, modifier = Modifier.semantics { heading() })
        chart()
    }
}

/** Horizontal bars: a name, a bar proportional to the value, and the value. */
@Composable
private fun Buckets(list: List<StatsBucket>, byHours: Boolean, unit: String) {
    val values = list.map { if (byHours && it.hours > 0) it.hours else it.titles.toDouble() }
    val max = (values.maxOrNull() ?: 1.0).coerceAtLeast(0.0001)
    Column(verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm)) {
        list.forEachIndexed { i, item ->
            Row(horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm), verticalAlignment = Alignment.CenterVertically) {
                Text(item.name, fontSize = Tokens.FontSize.footnote, color = Tokens.palette.fg, maxLines = 1, overflow = TextOverflow.Ellipsis, modifier = Modifier.width(96.dp))
                Box(Modifier.weight(1f).height(18.dp)) {
                    Box(Modifier.fillMaxHeight().fillMaxWidth((values[i] / max).toFloat().coerceIn(0.02f, 1f)).clip(RoundedCornerShape(4.dp)).background(Tokens.palette.primary))
                }
                Text(
                    if (byHours && item.hours > 0) StatsFormat.hours(item.hours, unit) else stringResource(R.string.stats_n_titles, item.titles.toString()),
                    fontSize = 11.sp, color = Tokens.palette.mutedFg, maxLines = 1,
                )
            }
        }
    }
}

/** Vertical bars with a label under each. */
@Composable
private fun Columns(items: List<Pair<String, Double>>) {
    val max = (items.maxOfOrNull { it.second } ?: 1.0).coerceAtLeast(0.0001)
    Row(Modifier.fillMaxWidth().height(150.dp), horizontalArrangement = Arrangement.spacedBy(3.dp), verticalAlignment = Alignment.Bottom) {
        items.forEach { (label, value) ->
            Column(Modifier.weight(1f).fillMaxHeight(), horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.Bottom) {
                Box(Modifier.weight(1f).fillMaxWidth(), contentAlignment = Alignment.BottomCenter) {
                    Box(Modifier.fillMaxWidth().fillMaxHeight((value / max).toFloat().coerceIn(0.01f, 1f)).clip(RoundedCornerShape(topStart = 3.dp, topEnd = 3.dp)).background(Tokens.palette.primary))
                }
                Text(label, fontSize = 9.sp, color = Tokens.palette.mutedFg, maxLines = 1)
            }
        }
    }
}
