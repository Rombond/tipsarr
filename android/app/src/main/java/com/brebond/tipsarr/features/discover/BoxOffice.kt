package com.brebond.tipsarr.features.discover

import androidx.compose.foundation.background
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
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.ArrowDropDown
import androidx.compose.material.icons.outlined.Tune
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.Text
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
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.heading
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.brebond.tipsarr.R
import com.brebond.tipsarr.core.api.ApiError
import com.brebond.tipsarr.core.api.MediaItem
import com.brebond.tipsarr.core.api.TipsarrApi
import com.brebond.tipsarr.core.api.boxOffice
import com.brebond.tipsarr.core.images.RemoteImage
import com.brebond.tipsarr.core.images.TmdbSize
import com.brebond.tipsarr.core.api.BoxOfficeChart
import com.brebond.tipsarr.core.api.BoxOfficeEntry
import com.brebond.tipsarr.design.Tokens
import com.brebond.tipsarr.design.palette
import com.brebond.tipsarr.ui.MediaPoster
import com.brebond.tipsarr.ui.StatusBadge
import com.brebond.tipsarr.ui.TopBarScaffold
import kotlinx.coroutines.launch

/** The latest chart for the person's region (Discover "For you"). */
class BoxOfficeModel(private val api: TipsarrApi) {
    var chart by mutableStateOf<BoxOfficeChart?>(null)
        private set

    suspend fun loadIfNeeded() { if (chart == null) refresh() }

    suspend fun refresh() {
        try { chart = api.boxOffice() } catch (_: ApiError) { /* the row is optional: no chart, no row */ }
    }
}

/** Ranked row on Discover: the chart's matched titles with the weekend gross, and a link to the full chart. */
@Composable
fun BoxOfficeRail(model: BoxOfficeModel, onOpenItem: (MediaItem) -> Unit, onOpenChart: (BoxOfficeChart) -> Unit) {
    LaunchedEffect(model) { model.loadIfNeeded() }
    val chart = model.chart ?: return
    val entries = chart.entries.filter { it.item != null }
    if (entries.isEmpty()) return
    Column(verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.md)) {
        Row(Modifier.fillMaxWidth().padding(horizontal = Tokens.Spacing.lg), verticalAlignment = Alignment.CenterVertically) {
            Column(Modifier.weight(1f)) {
                Text(stringResource(R.string.nav_boxoffice), fontSize = Tokens.FontSize.title3, fontWeight = FontWeight.SemiBold, color = Tokens.palette.fg, modifier = Modifier.semantics { heading() })
                Text(chart.title(), fontSize = Tokens.FontSize.footnote, color = Tokens.palette.mutedFg)
            }
            Text(
                stringResource(R.string.m_boxoffice_full), fontSize = Tokens.FontSize.subhead, color = Tokens.palette.mutedFg,
                modifier = Modifier.clickable { onOpenChart(chart) }.padding(Tokens.Spacing.sm),
            )
        }
        LazyRow(horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.md), contentPadding = PaddingValues(horizontal = Tokens.Spacing.lg)) {
            items(entries, key = { it.position }) { entry ->
                val item = entry.item ?: return@items
                MediaPoster(item, { onOpenItem(item) }, Modifier.width(120.dp), subtitle = chart.money(entry.weekendGross), rank = entry.position)
            }
        }
    }
}

/** The whole chart as a list; the region and week can be changed. */
@Composable
fun BoxOfficeScreen(api: TipsarrApi, first: BoxOfficeChart, onBack: () -> Unit, onOpenItem: (MediaItem) -> Unit) {
    var chart by remember { mutableStateOf(first) }
    var menu by remember { mutableStateOf(false) }
    val scope = rememberCoroutineScope()
    fun load(region: String, week: String?) {
        scope.launch { try { chart = api.boxOffice(region, week) } catch (_: ApiError) {} }
    }
    TopBarScaffold(
        stringResource(R.string.nav_boxoffice), onBack,
        actions = {
            Box {
                IconButton({ menu = true }) { Icon(Icons.Outlined.Tune, stringResource(R.string.m_boxoffice_region), tint = Tokens.palette.fg) }
                DropdownMenu(menu, { menu = false }, containerColor = Tokens.palette.card) {
                    Text(stringResource(R.string.m_boxoffice_region), Modifier.padding(horizontal = Tokens.Spacing.md), fontSize = Tokens.FontSize.footnote, color = Tokens.palette.mutedFg)
                    chart.regions.forEach { region ->
                        DropdownMenuItem(text = { Text(region, color = Tokens.palette.fg, fontWeight = if (region == chart.region) FontWeight.Bold else FontWeight.Normal) }, onClick = { menu = false; load(region, null) })
                    }
                    Text(stringResource(R.string.m_boxoffice_week), Modifier.padding(horizontal = Tokens.Spacing.md, vertical = Tokens.Spacing.xs), fontSize = Tokens.FontSize.footnote, color = Tokens.palette.mutedFg)
                    chart.weeks.take(12).forEach { week ->
                        DropdownMenuItem(text = { Text(week.label, color = Tokens.palette.fg, fontWeight = if (week.key == chart.week) FontWeight.Bold else FontWeight.Normal) }, onClick = { menu = false; load(chart.region, week.key) })
                    }
                }
            }
        },
    ) {
        LazyColumn(Modifier.fillMaxSize(), contentPadding = PaddingValues(horizontal = Tokens.Spacing.lg)) {
            item { Text(chart.title(), fontSize = Tokens.FontSize.subhead, color = Tokens.palette.mutedFg, modifier = Modifier.padding(vertical = Tokens.Spacing.md)) }
            if (chart.entries.isEmpty()) item { Text(stringResource(R.string.m_boxoffice_empty), color = Tokens.palette.mutedFg) }
            items(chart.entries, key = { it.position }) { entry -> ChartRow(chart, entry, onOpenItem) }
        }
    }
}

@Composable
private fun ChartRow(chart: BoxOfficeChart, entry: BoxOfficeEntry, onOpenItem: (MediaItem) -> Unit) {
    val item = entry.item
    Row(
        Modifier.fillMaxWidth().then(if (item != null) Modifier.clickable { onOpenItem(item) } else Modifier).padding(vertical = Tokens.Spacing.sm),
        horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.md), verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(entry.position.toString(), fontSize = Tokens.FontSize.title3, fontWeight = FontWeight.Bold, color = Tokens.palette.fg, modifier = Modifier.width(28.dp))
        Box(Modifier.width(48.dp).height(72.dp).clip(RoundedCornerShape(Tokens.Radius.sm)).background(Tokens.palette.muted)) {
            RemoteImage(item?.posterPath, TmdbSize.W92, Modifier.fillMaxSize())
        }
        Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
            Text(item?.title ?: entry.title, fontSize = Tokens.FontSize.body, fontWeight = FontWeight.SemiBold, color = Tokens.palette.fg, maxLines = 2, overflow = TextOverflow.Ellipsis)
            Text(chart.money(entry.weekendGross), fontSize = Tokens.FontSize.footnote, color = Tokens.palette.mutedFg)
            Text("${chart.money(entry.totalGross)} · ${stringResource(R.string.m_boxoffice_weeks, entry.weeksInRelease.toString())}", fontSize = Tokens.FontSize.caption, color = Tokens.palette.mutedFg)
        }
        item?.state?.let { StatusBadge(it, compact = true) }
    }
}
