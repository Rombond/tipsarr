package com.brebond.tipsarr.features.library

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
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
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import com.brebond.tipsarr.R
import com.brebond.tipsarr.core.api.LibraryFacets
import com.brebond.tipsarr.core.api.LibraryFilters
import com.brebond.tipsarr.core.api.LibrarySort
import com.brebond.tipsarr.core.api.WatchedFilter
import com.brebond.tipsarr.design.Tokens
import com.brebond.tipsarr.design.palette
import com.brebond.tipsarr.ui.FilterMinRating
import com.brebond.tipsarr.ui.FilterSection
import com.brebond.tipsarr.ui.FilterSheet
import com.brebond.tipsarr.ui.FilterSort
import com.brebond.tipsarr.ui.FilterYearRange
import com.brebond.tipsarr.ui.GenrePicker

private fun LibrarySort.title(): Int = when (this) {
    LibrarySort.Added -> R.string.library_sort_added
    LibrarySort.Title -> R.string.library_sort_title
    LibrarySort.Year -> R.string.library_sort_year
    LibrarySort.Rating -> R.string.library_sort_rating
    LibrarySort.Runtime -> R.string.library_sort_runtime
    LibrarySort.Popular -> R.string.library_sort_popular
}

/** Genres, year range, minimum rating, runtime, watched state and sort order. Changes apply as you make them. */
@Composable
fun LibraryFiltersSheet(initial: LibraryFilters, facets: LibraryFacets?, onDismiss: () -> Unit, onChange: (LibraryFilters) -> Unit) {
    var filters by remember { mutableStateOf(initial) }
    fun update(new: LibraryFilters) { filters = new; onChange(new) }

    FilterSheet(
        stringResource(R.string.library_filters),
        canReset = !(filters.sheetIsDefault && filters.watched == WatchedFilter.Any),
        onReset = { update(LibraryFilters(kind = filters.kind, query = filters.query)) },
        onDismiss = onDismiss,
    ) {
        if (facets != null && facets.genres.isNotEmpty()) FilterSection(stringResource(R.string.library_genres)) {
            GenrePicker(
                facets.genres.map { it.name }, filters.genres, { update(filters.copy(genres = it)) },
                filters.anyGenre, { update(filters.copy(anyGenre = it)) },
            )
        }
        if (facets != null && facets.yearMax >= facets.yearMin && facets.yearMax > 0) {
            FilterYearRange(
                filters.yearFrom, filters.yearTo, (facets.yearMax downTo facets.yearMin).toList(),
                { update(filters.copy(yearFrom = it)) }, { update(filters.copy(yearTo = it)) },
            )
        }
        FilterMinRating(filters.minRating) { update(filters.copy(minRating = it)) }
        if (facets != null && facets.maxRuntimeMinutes > 20) FilterSection(stringResource(R.string.library_runtime)) {
            Column(verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.xs)) {
                val max = facets.maxRuntimeMinutes.toFloat()
                Slider(
                    value = (filters.maxRuntime ?: facets.maxRuntimeMinutes).toFloat().coerceIn(20f, max), valueRange = 20f..max,
                    steps = ((facets.maxRuntimeMinutes - 20) / 5 - 1).coerceAtLeast(0),
                    onValueChange = { v -> update(filters.copy(maxRuntime = if (v.toInt() >= facets.maxRuntimeMinutes - 2) null else (v / 5).toInt() * 5)) },
                    colors = SliderDefaults.colors(thumbColor = Tokens.palette.primary, activeTrackColor = Tokens.palette.primary, inactiveTrackColor = Tokens.palette.muted),
                )
                Text(filters.maxRuntime?.let { stringResource(R.string.m_library_under, hoursText(it)) } ?: stringResource(R.string.library_any), fontSize = Tokens.FontSize.footnote, color = Tokens.palette.mutedFg)
            }
        }
        FilterSection(stringResource(R.string.library_watched_state)) {
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
        // Titles are resolved here because FilterSort takes plain strings.
        val titles = LibrarySort.entries.associateWith { stringResource(it.title()) }
        FilterSort(filters.sort, LibrarySort.entries, { titles.getValue(it) }, { update(filters.copy(sort = it)) }) {
            SingleChoiceSegmentedButtonRow {
                SegmentedButton(selected = filters.descending, onClick = { update(filters.copy(descending = true)) }, shape = SegmentedButtonDefaults.itemShape(0, 2)) { Text(stringResource(R.string.library_descending)) }
                SegmentedButton(selected = !filters.descending, onClick = { update(filters.copy(descending = false)) }, shape = SegmentedButtonDefaults.itemShape(1, 2)) { Text(stringResource(R.string.library_ascending)) }
            }
        }
    }
}

/** Minutes as "1h 45m" (or "45m"): the slider reaches several hours in big libraries. */
private fun hoursText(minutes: Int): String = if (minutes >= 60) "${minutes / 60}h" + (minutes % 60).takeIf { it > 0 }?.let { " ${it}m" }.orEmpty() else "${minutes}m"
