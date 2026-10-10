package com.brebond.tipsarr.features.library

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.defaultMinSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.KeyboardArrowDown
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.SegmentedButton
import androidx.compose.material3.SegmentedButtonDefaults
import androidx.compose.material3.SingleChoiceSegmentedButtonRow
import androidx.compose.material3.Slider
import androidx.compose.material3.SliderDefaults
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.heading
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.brebond.tipsarr.R
import com.brebond.tipsarr.core.api.LibraryFacets
import com.brebond.tipsarr.core.api.LibraryFilters
import com.brebond.tipsarr.core.api.LibrarySort
import com.brebond.tipsarr.core.api.WatchedFilter
import com.brebond.tipsarr.design.Tokens
import com.brebond.tipsarr.design.palette
import com.brebond.tipsarr.ui.ButtonKind
import com.brebond.tipsarr.ui.Chip
import com.brebond.tipsarr.ui.TipsarrButton
import com.brebond.tipsarr.ui.TipsarrSheet
import java.util.Locale

private fun LibrarySort.title(): Int = when (this) {
    LibrarySort.Added -> R.string.library_sort_added
    LibrarySort.Title -> R.string.library_sort_title
    LibrarySort.Year -> R.string.library_sort_year
    LibrarySort.Rating -> R.string.library_sort_rating
    LibrarySort.Runtime -> R.string.library_sort_runtime
    LibrarySort.Popular -> R.string.library_sort_popular
}

/** Genres, year range, minimum rating, runtime, watched state and sort order. Changes apply as you make them. */
@OptIn(ExperimentalLayoutApi::class, ExperimentalMaterial3Api::class)
@Composable
fun LibraryFiltersSheet(initial: LibraryFilters, facets: LibraryFacets?, onDismiss: () -> Unit, onChange: (LibraryFilters) -> Unit) {
    var filters by remember { mutableStateOf(initial) }
    fun update(new: LibraryFilters) { filters = new; onChange(new) }

    TipsarrSheet(onDismiss, fullHeight = true) {
        Column(
            Modifier.fillMaxWidth().verticalScroll(rememberScrollState()).navigationBarsPadding().padding(horizontal = Tokens.Spacing.xl).padding(bottom = Tokens.Spacing.xl),
            verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.x2xl),
        ) {
            Text(stringResource(R.string.library_filters), fontSize = Tokens.FontSize.title3, fontWeight = FontWeight.SemiBold, color = Tokens.palette.fg, modifier = Modifier.semantics { heading() })
            if (facets != null && facets.genres.isNotEmpty()) Section(stringResource(R.string.library_genres)) {
                FlowRow(horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm), verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm)) {
                    facets.genres.take(30).forEach { genre ->
                        val selected = genre.name in filters.genres
                        Chip(genre.name, selected = selected, onClick = {
                            if (selected) update(filters.copy(genres = filters.genres - genre.name))
                            else if (filters.genres.size < 6) update(filters.copy(genres = filters.genres + genre.name))
                        })
                    }
                }
                val modes = listOf(false to R.string.m_library_genre_all, true to R.string.m_library_genre_any)
                SingleChoiceSegmentedButtonRow(Modifier.fillMaxWidth()) {
                    modes.forEachIndexed { index, (any, label) ->
                        SegmentedButton(
                            selected = filters.anyGenre == any, onClick = { update(filters.copy(anyGenre = any)) },
                            shape = SegmentedButtonDefaults.itemShape(index, modes.size),
                        ) { Text(stringResource(label)) }
                    }
                }
            }
            if (facets != null && facets.yearMax >= facets.yearMin && facets.yearMax > 0) Section(stringResource(R.string.library_year)) {
                Row(horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.md)) {
                    val years = (facets.yearMax downTo facets.yearMin).toList()
                    Box(Modifier.weight(1f)) { YearMenu(stringResource(R.string.library_year_from), filters.yearFrom, years) { update(filters.copy(yearFrom = it)) } }
                    Box(Modifier.weight(1f)) { YearMenu(stringResource(R.string.library_year_to), filters.yearTo, years) { update(filters.copy(yearTo = it)) } }
                }
            }
            Section(stringResource(R.string.library_min_rating)) {
                Slider(
                    value = (filters.minRating ?: 0.0).toFloat(), valueRange = 0f..9f, steps = 17,
                    onValueChange = { v -> update(filters.copy(minRating = if (v < 0.5f) null else (v * 2).toInt() / 2.0)) },
                    colors = SliderDefaults.colors(thumbColor = Tokens.palette.primary, activeTrackColor = Tokens.palette.primary, inactiveTrackColor = Tokens.palette.muted),
                )
                Text(filters.minRating?.let { String.format(Locale.getDefault(), "%.1f", it) } ?: stringResource(R.string.library_any), fontSize = Tokens.FontSize.footnote, color = Tokens.palette.mutedFg)
            }
            if (facets != null && facets.maxRuntimeMinutes > 20) Section(stringResource(R.string.library_runtime)) {
                val max = facets.maxRuntimeMinutes.toFloat()
                Slider(
                    value = (filters.maxRuntime ?: facets.maxRuntimeMinutes).toFloat().coerceIn(20f, max), valueRange = 20f..max,
                    steps = ((facets.maxRuntimeMinutes - 20) / 5 - 1).coerceAtLeast(0),
                    onValueChange = { v -> update(filters.copy(maxRuntime = if (v.toInt() >= facets.maxRuntimeMinutes - 2) null else (v / 5).toInt() * 5)) },
                    colors = SliderDefaults.colors(thumbColor = Tokens.palette.primary, activeTrackColor = Tokens.palette.primary, inactiveTrackColor = Tokens.palette.muted),
                )
                Text(filters.maxRuntime?.let { stringResource(R.string.m_library_under, hoursText(it)) } ?: stringResource(R.string.library_any), fontSize = Tokens.FontSize.footnote, color = Tokens.palette.mutedFg)
            }
            Section(stringResource(R.string.library_watched_state)) {
                val options = listOf(WatchedFilter.Any to R.string.library_any, WatchedFilter.Yes to R.string.library_watched, WatchedFilter.No to R.string.library_not_watched)
                SingleChoiceSegmentedButtonRow(Modifier.fillMaxWidth()) {
                    options.forEachIndexed { index, (value, label) ->
                        SegmentedButton(
                            selected = filters.watched == value, onClick = { update(filters.copy(watched = value)) },
                            shape = SegmentedButtonDefaults.itemShape(index, options.size),
                        ) { Text(stringResource(label)) }
                    }
                }
            }
            Section(stringResource(R.string.library_sort_by)) {
                Row(horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.md), verticalAlignment = Alignment.CenterVertically) {
                    var menu by remember { mutableStateOf(false) }
                    Box(Modifier.weight(1f)) {
                        MenuField(stringResource(filters.sort.title())) { menu = true }
                        DropdownMenu(menu, { menu = false }, containerColor = Tokens.palette.card) {
                            LibrarySort.entries.forEach { sort ->
                                DropdownMenuItem(text = { Text(stringResource(sort.title()), color = Tokens.palette.fg) }, onClick = { menu = false; update(filters.copy(sort = sort)) })
                            }
                        }
                    }
                    SingleChoiceSegmentedButtonRow {
                        SegmentedButton(selected = filters.descending, onClick = { update(filters.copy(descending = true)) }, shape = SegmentedButtonDefaults.itemShape(0, 2)) { Text(stringResource(R.string.library_descending)) }
                        SegmentedButton(selected = !filters.descending, onClick = { update(filters.copy(descending = false)) }, shape = SegmentedButtonDefaults.itemShape(1, 2)) { Text(stringResource(R.string.library_ascending)) }
                    }
                }
            }
            Row(horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.md)) {
                Box(Modifier.weight(1f)) {
                    TipsarrButton(
                        stringResource(R.string.library_reset),
                        { update(LibraryFilters(kind = filters.kind, query = filters.query)) },
                        fullWidth = true, kind = ButtonKind.Secondary, enabled = !(filters.sheetIsDefault && filters.watched == WatchedFilter.Any),
                    )
                }
                Box(Modifier.weight(1f)) { TipsarrButton(stringResource(R.string.m_library_apply), onDismiss, fullWidth = true) }
            }
        }
    }
}

@Composable
private fun Section(title: String, content: @Composable () -> Unit) {
    Column(verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.md)) {
        Text(title, fontSize = Tokens.FontSize.headline, fontWeight = FontWeight.SemiBold, color = Tokens.palette.fg)
        content()
    }
}

@Composable
private fun MenuField(text: String, onClick: () -> Unit) {
    Row(
        Modifier.fillMaxWidth().defaultMinSize(minHeight = Tokens.Size.touchTarget).clip(RoundedCornerShape(Tokens.Radius.md))
            .background(Tokens.palette.muted).clickable(onClick = onClick).padding(horizontal = Tokens.Spacing.md),
        horizontalArrangement = Arrangement.SpaceBetween, verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(text, fontSize = Tokens.FontSize.body, color = Tokens.palette.fg)
        Icon(Icons.Outlined.KeyboardArrowDown, null, tint = Tokens.palette.mutedFg)
    }
}

@Composable
private fun YearMenu(label: String, selected: Int?, years: List<Int>, onSelect: (Int?) -> Unit) {
    var menu by remember { mutableStateOf(false) }
    Box {
        MenuField(selected?.toString() ?: label) { menu = true }
        DropdownMenu(menu, { menu = false }, containerColor = Tokens.palette.card, modifier = Modifier.defaultMinSize(minWidth = 120.dp)) {
            DropdownMenuItem(text = { Text(stringResource(R.string.library_any), color = Tokens.palette.fg) }, onClick = { menu = false; onSelect(null) })
            years.forEach { year -> DropdownMenuItem(text = { Text(year.toString(), color = Tokens.palette.fg) }, onClick = { menu = false; onSelect(year) }) }
        }
    }
}

/** Minutes as "1h 45m" (or "45m"): the slider reaches several hours in big libraries. */
private fun hoursText(minutes: Int): String = if (minutes >= 60) "${minutes / 60}h" + (minutes % 60).takeIf { it > 0 }?.let { " ${it}m" }.orEmpty() else "${minutes}m"
