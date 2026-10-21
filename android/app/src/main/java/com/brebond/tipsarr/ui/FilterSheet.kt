package com.brebond.tipsarr.ui

import androidx.compose.animation.AnimatedVisibility
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.gestures.snapping.rememberSnapFlingBehavior
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.defaultMinSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyListState
import androidx.compose.foundation.lazy.itemsIndexed
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.Check
import androidx.compose.material.icons.outlined.Close
import androidx.compose.material.icons.outlined.KeyboardArrowDown
import androidx.compose.material.icons.outlined.Refresh
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.Slider
import androidx.compose.material3.SliderDefaults
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.derivedStateOf
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.rememberUpdatedState
import androidx.compose.runtime.setValue
import androidx.compose.runtime.snapshotFlow
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.semantics.heading
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import com.brebond.tipsarr.R
import com.brebond.tipsarr.design.Tokens
import com.brebond.tipsarr.design.palette
import kotlinx.coroutines.flow.filter
import kotlinx.coroutines.launch
import java.util.Locale
import kotlin.math.abs

/**
 * Building blocks of the Library and Search filter sheets, so the two sheets look and behave the same.
 * Changes apply as you make them: close and apply both dismiss the sheet.
 */

/** Sheet with close and reset icons on the left, the title, and an apply check on the right. */
@Composable
fun FilterSheet(title: String, canReset: Boolean, onReset: () -> Unit, onDismiss: () -> Unit, content: @Composable ColumnScope.() -> Unit) {
    TipsarrSheet(onDismiss, fullHeight = true) {
        Row(Modifier.fillMaxWidth().padding(horizontal = Tokens.Spacing.sm), verticalAlignment = Alignment.CenterVertically) {
            IconButton(onDismiss) { Icon(Icons.Outlined.Close, stringResource(R.string.m_library_close), tint = Tokens.palette.fg) }
            IconButton(onReset, enabled = canReset) {
                Icon(Icons.Outlined.Refresh, stringResource(R.string.library_reset), tint = if (canReset) Tokens.palette.fg else Tokens.palette.mutedFg)
            }
            Text(
                title, Modifier.weight(1f).semantics { heading() }, textAlign = TextAlign.Center,
                fontSize = Tokens.FontSize.headline, fontWeight = FontWeight.SemiBold, color = Tokens.palette.fg,
            )
            // Balances the two icons on the left so the title stays centred.
            Spacer(Modifier.size(48.dp))
            IconButton(onDismiss) { Icon(Icons.Outlined.Check, stringResource(R.string.m_library_apply), tint = Tokens.palette.primary) }
        }
        Column(
            Modifier.fillMaxWidth().verticalScroll(rememberScrollState()).navigationBarsPadding()
                .padding(horizontal = Tokens.Spacing.xl).padding(top = Tokens.Spacing.md, bottom = Tokens.Spacing.xl),
            verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.x2xl),
        ) { content() }
    }
}

@Composable
fun FilterSection(title: String, content: @Composable () -> Unit) {
    Column(verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.md)) {
        Text(title, fontSize = Tokens.FontSize.headline, fontWeight = FontWeight.SemiBold, color = Tokens.palette.fg, modifier = Modifier.semantics { heading() })
        content()
    }
}

@Composable
fun FilterMenuField(text: String, onClick: () -> Unit) {
    Row(
        Modifier.fillMaxWidth().defaultMinSize(minHeight = Tokens.Size.touchTarget).clip(RoundedCornerShape(Tokens.Radius.md))
            .background(Tokens.palette.muted).clickable(role = Role.Button, onClick = onClick).padding(horizontal = Tokens.Spacing.md),
        horizontalArrangement = Arrangement.SpaceBetween, verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(text, fontSize = Tokens.FontSize.body, color = Tokens.palette.fg)
        Icon(Icons.Outlined.KeyboardArrowDown, null, tint = Tokens.palette.mutedFg)
    }
}

private enum class YearField { From, To }

/** Two rows ("from", "to") in one card; tapping a row unfolds a wheel under it, like the start and end times in Calendar. */
@Composable
fun FilterYearRange(from: Int?, to: Int?, years: List<Int>, onFromChange: (Int?) -> Unit, onToChange: (Int?) -> Unit) {
    var open by remember { mutableStateOf<YearField?>(null) }
    FilterSection(stringResource(R.string.library_year)) {
        Column(Modifier.fillMaxWidth().clip(RoundedCornerShape(Tokens.Radius.md)).background(Tokens.palette.muted)) {
            YearRow(stringResource(R.string.library_year_from), from, open == YearField.From) { open = if (open == YearField.From) null else YearField.From }
            AnimatedVisibility(open == YearField.From) { YearWheel(years, from, onFromChange) }
            HorizontalDivider(Modifier.padding(start = Tokens.Spacing.md), color = Tokens.palette.border)
            YearRow(stringResource(R.string.library_year_to), to, open == YearField.To) { open = if (open == YearField.To) null else YearField.To }
            AnimatedVisibility(open == YearField.To) { YearWheel(years, to, onToChange) }
        }
    }
}

@Composable
private fun YearRow(title: String, year: Int?, open: Boolean, onClick: () -> Unit) {
    Row(
        Modifier.fillMaxWidth().defaultMinSize(minHeight = Tokens.Size.touchTarget).clickable(role = Role.Button, onClick = onClick).padding(horizontal = Tokens.Spacing.md),
        horizontalArrangement = Arrangement.SpaceBetween, verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(title, fontSize = Tokens.FontSize.body, color = Tokens.palette.fg)
        Text(year?.toString() ?: stringResource(R.string.library_any), fontSize = Tokens.FontSize.body, color = if (open) Tokens.palette.primary else Tokens.palette.mutedFg)
    }
}

private val WHEEL_ROW = 38.dp
private val WHEEL_HEIGHT = 150.dp

/** "Any" then the years, newest first. The row in the middle is the choice; it is applied when the wheel stops. */
@Composable
private fun YearWheel(years: List<Int>, selected: Int?, onSelect: (Int?) -> Unit) {
    val options = remember(years) { listOf<Int?>(null) + years }
    val state = rememberLazyListState(remember { options.indexOf(selected).coerceAtLeast(0) })
    val current by rememberUpdatedState(selected)
    val onSelectNow by rememberUpdatedState(onSelect)
    val centered by remember { derivedStateOf { centeredIndex(state) } }
    val scope = rememberCoroutineScope()
    LaunchedEffect(state, options) {
        snapshotFlow { state.isScrollInProgress }.filter { !it }.collect {
            val index = centeredIndex(state)
            if (index in options.indices && options[index] != current) onSelectNow(options[index])
        }
    }
    Box(Modifier.fillMaxWidth().height(WHEEL_HEIGHT), contentAlignment = Alignment.Center) {
        Box(Modifier.fillMaxWidth().padding(horizontal = Tokens.Spacing.sm).height(WHEEL_ROW).clip(RoundedCornerShape(Tokens.Radius.sm)).background(Tokens.palette.border))
        LazyColumn(
            Modifier.fillMaxWidth().height(WHEEL_HEIGHT), state = state, flingBehavior = rememberSnapFlingBehavior(state),
            contentPadding = androidx.compose.foundation.layout.PaddingValues(vertical = (WHEEL_HEIGHT - WHEEL_ROW) / 2),
        ) {
            itemsIndexed(options) { index, year ->
                Box(Modifier.fillMaxWidth().height(WHEEL_ROW).clickable { scope.launch { state.animateScrollToItem(index) } }, contentAlignment = Alignment.Center) {
                    Text(
                        year?.toString() ?: stringResource(R.string.library_any), fontSize = Tokens.FontSize.body,
                        fontWeight = if (index == centered) FontWeight.SemiBold else FontWeight.Normal,
                        color = if (index == centered) Tokens.palette.fg else Tokens.palette.mutedFg,
                    )
                }
            }
        }
    }
}

private fun centeredIndex(state: LazyListState): Int {
    val info = state.layoutInfo
    val middle = (info.viewportStartOffset + info.viewportEndOffset) / 2
    return info.visibleItemsInfo.minByOrNull { abs(it.offset + it.size / 2 - middle) }?.index ?: state.firstVisibleItemIndex
}

@Composable
fun FilterMinRating(rating: Double?, onChange: (Double?) -> Unit) {
    FilterSection(stringResource(R.string.library_min_rating)) {
        Column(verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.xs)) {
            Slider(
                value = (rating ?: 0.0).toFloat(), valueRange = 0f..9f, steps = 17,
                onValueChange = { v -> onChange(if (v < 0.5f) null else (v * 2).toInt() / 2.0) },
                colors = SliderDefaults.colors(thumbColor = Tokens.palette.primary, activeTrackColor = Tokens.palette.primary, inactiveTrackColor = Tokens.palette.muted),
            )
            Text(rating?.let { String.format(Locale.getDefault(), "%.1f", it) } ?: stringResource(R.string.library_any), fontSize = Tokens.FontSize.footnote, color = Tokens.palette.mutedFg)
        }
    }
}

/** Sort menu; `trailing` holds an extra control next to it (the Library's ascending/descending switch). */
@Composable
fun <Option> FilterSort(selection: Option, options: List<Option>, title: (Option) -> String, onChange: (Option) -> Unit, trailing: @Composable () -> Unit = {}) {
    FilterSection(stringResource(R.string.library_sort_by)) {
        Row(horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.md), verticalAlignment = Alignment.CenterVertically) {
            var menu by remember { mutableStateOf(false) }
            Box(Modifier.weight(1f)) {
                FilterMenuField(title(selection)) { menu = true }
                DropdownMenu(menu, { menu = false }, containerColor = Tokens.palette.card) {
                    options.forEach { option ->
                        DropdownMenuItem(text = { Text(title(option), color = Tokens.palette.fg) }, onClick = { menu = false; onChange(option) })
                    }
                }
            }
            trailing()
        }
    }
}
