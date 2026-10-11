package com.brebond.tipsarr.features.search

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
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
import androidx.compose.ui.semantics.heading
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import com.brebond.tipsarr.R
import com.brebond.tipsarr.design.Tokens
import com.brebond.tipsarr.design.palette
import com.brebond.tipsarr.features.library.MenuField
import com.brebond.tipsarr.features.library.Section
import com.brebond.tipsarr.features.library.YearMenu
import com.brebond.tipsarr.ui.ButtonKind
import com.brebond.tipsarr.ui.GenrePicker
import com.brebond.tipsarr.ui.TipsarrButton
import com.brebond.tipsarr.ui.TipsarrSheet
import java.util.Calendar
import java.util.Locale

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
    androidx.compose.runtime.LaunchedEffect(Unit) { model.loadGenres() }
    val thisYear = remember { Calendar.getInstance().get(Calendar.YEAR) + 1 }
    val years = remember(thisYear) { (thisYear downTo 1900).toList() }

    TipsarrSheet(onDismiss, fullHeight = true) {
        Column(
            Modifier.fillMaxWidth().verticalScroll(rememberScrollState()).navigationBarsPadding().padding(horizontal = Tokens.Spacing.xl).padding(bottom = Tokens.Spacing.xl),
            verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.x2xl),
        ) {
            Text(stringResource(R.string.m_search_filters), fontSize = Tokens.FontSize.title3, fontWeight = FontWeight.SemiBold, color = Tokens.palette.fg, modifier = Modifier.semantics { heading() })
            if (model.genres.isNotEmpty()) Section(stringResource(R.string.library_genres)) {
                GenrePicker(model.genres.map { it.name }, filters.genres, { update(filters.copy(genres = it)) })
            }
            Section(stringResource(R.string.library_year)) {
                Row(horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.md)) {
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
            Section(stringResource(R.string.m_search_avail)) {
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
            Section(stringResource(R.string.library_sort_by)) {
                var menu by remember { mutableStateOf(false) }
                Box {
                    MenuField(stringResource(filters.sort.title())) { menu = true }
                    DropdownMenu(menu, { menu = false }, containerColor = Tokens.palette.card) {
                        SearchSort.entries.forEach { sort ->
                            DropdownMenuItem(text = { Text(stringResource(sort.title()), color = Tokens.palette.fg) }, onClick = { menu = false; update(filters.copy(sort = sort)) })
                        }
                    }
                }
            }
            Row(horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.md)) {
                Box(Modifier.weight(1f)) {
                    TipsarrButton(stringResource(R.string.library_reset), { update(SearchFilters()) }, fullWidth = true, kind = ButtonKind.Secondary, enabled = filters.activeCount > 0)
                }
                Box(Modifier.weight(1f)) { TipsarrButton(stringResource(R.string.m_library_apply), onDismiss, fullWidth = true) }
            }
        }
    }
}
