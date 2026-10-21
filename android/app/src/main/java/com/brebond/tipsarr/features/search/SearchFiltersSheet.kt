package com.brebond.tipsarr.features.search

import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.material3.SegmentedButton
import androidx.compose.material3.SegmentedButtonDefaults
import androidx.compose.material3.SingleChoiceSegmentedButtonRow
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import com.brebond.tipsarr.R
import com.brebond.tipsarr.ui.FilterMinRating
import com.brebond.tipsarr.ui.FilterSection
import com.brebond.tipsarr.ui.FilterSheet
import com.brebond.tipsarr.ui.FilterSort
import com.brebond.tipsarr.ui.FilterYearRange
import com.brebond.tipsarr.ui.GenrePicker
import java.util.Calendar

private fun SearchSort.title(): Int = when (this) {
    SearchSort.Relevance -> R.string.m_search_sort_relevance
    SearchSort.Rating -> R.string.library_sort_rating
    SearchSort.Year -> R.string.library_sort_year
}

private fun SearchAvailability.title(): Int = when (this) {
    SearchAvailability.Any -> R.string.library_any
    SearchAvailability.Missing -> R.string.m_search_avail_missing
    SearchAvailability.InLibrary -> R.string.m_search_avail_in
}

/** Genres, year range, minimum rating, availability and sort for the search results. Changes apply as you make them. */
@Composable
fun SearchFiltersSheet(model: SearchModel, onDismiss: () -> Unit) {
    var filters by remember { mutableStateOf(model.filters) }
    fun update(new: SearchFilters) { filters = new; model.filters = new }
    LaunchedEffect(Unit) { model.loadGenres() }
    val thisYear = remember { Calendar.getInstance().get(Calendar.YEAR) + 1 }
    val years = remember(thisYear) { (thisYear downTo 1900).toList() }

    FilterSheet(stringResource(R.string.m_search_filters), canReset = filters.activeCount > 0, onReset = { update(SearchFilters()) }, onDismiss = onDismiss) {
        if (model.genres.isNotEmpty()) FilterSection(stringResource(R.string.library_genres)) {
            GenrePicker(
                model.genres.map { it.name }, filters.genres, { update(filters.copy(genres = it)) },
                filters.anyGenre, { update(filters.copy(anyGenre = it)) },
            )
        }
        FilterYearRange(filters.yearFrom, filters.yearTo, years, { update(filters.copy(yearFrom = it)) }, { update(filters.copy(yearTo = it)) })
        FilterMinRating(filters.minRating) { update(filters.copy(minRating = it)) }
        FilterSection(stringResource(R.string.m_search_avail)) {
            val options = SearchAvailability.entries
            SingleChoiceSegmentedButtonRow(Modifier.fillMaxWidth()) {
                options.forEachIndexed { index, value ->
                    SegmentedButton(
                        selected = filters.availability == value, onClick = { update(filters.copy(availability = value)) },
                        shape = SegmentedButtonDefaults.itemShape(index, options.size),
                    ) { Text(stringResource(value.title())) }
                }
            }
        }
        // Titles are resolved here because FilterSort takes plain strings.
        val titles = SearchSort.entries.associateWith { stringResource(it.title()) }
        FilterSort(filters.sort, SearchSort.entries, { titles.getValue(it) }, { update(filters.copy(sort = it)) })
    }
}
