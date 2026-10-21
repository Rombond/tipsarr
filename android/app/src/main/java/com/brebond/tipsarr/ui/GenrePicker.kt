package com.brebond.tipsarr.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.defaultMinSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.KeyboardArrowDown
import androidx.compose.material.icons.outlined.Refresh
import androidx.compose.material.icons.outlined.Search
import androidx.compose.material3.Checkbox
import androidx.compose.material3.CheckboxDefaults
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.OutlinedTextFieldDefaults
import androidx.compose.material3.SegmentedButton
import androidx.compose.material3.SegmentedButtonDefaults
import androidx.compose.material3.SingleChoiceSegmentedButtonRow
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
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.semantics.heading
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import com.brebond.tipsarr.R
import com.brebond.tipsarr.design.Tokens
import com.brebond.tipsarr.design.palette

/**
 * One compact row that shows the chosen genres ("Any genre", "Action", "Action +2") and opens a searchable
 * multi-select list. Used by the Library and Search filters; at most `max` genres can be chosen.
 */
@Composable
fun GenrePicker(
    names: List<String>, selected: Set<String>, onChange: (Set<String>) -> Unit, anyGenre: Boolean, onAnyChange: (Boolean) -> Unit,
    modifier: Modifier = Modifier, max: Int = 6,
) {
    var open by remember { mutableStateOf(false) }
    val summary = when (selected.size) {
        0 -> stringResource(R.string.m_library_genres_any)
        1 -> selected.first()
        else -> stringResource(R.string.m_library_genres_more, selected.first(), selected.size - 1)
    }
    Row(
        modifier.fillMaxWidth().defaultMinSize(minHeight = Tokens.Size.touchTarget).clip(RoundedCornerShape(Tokens.Radius.md))
            .background(Tokens.palette.muted).clickable(role = Role.Button) { open = true }.padding(horizontal = Tokens.Spacing.md),
        horizontalArrangement = Arrangement.SpaceBetween, verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(summary, fontSize = Tokens.FontSize.body, color = Tokens.palette.fg, maxLines = 1, overflow = TextOverflow.Ellipsis, modifier = Modifier.weight(1f))
        Icon(Icons.Outlined.KeyboardArrowDown, null, tint = Tokens.palette.mutedFg)
    }
    if (open) GenreListSheet(names, selected, onChange, anyGenre, onAnyChange, max) { open = false }
}

@Composable
private fun GenreListSheet(
    names: List<String>, selected: Set<String>, onChange: (Set<String>) -> Unit, anyGenre: Boolean, onAnyChange: (Boolean) -> Unit,
    max: Int, onDismiss: () -> Unit,
) {
    var chosen by remember { mutableStateOf(selected) }
    var query by remember { mutableStateOf("") }
    val shown = remember(names, query) { names.filter { it.contains(query.trim(), ignoreCase = true) } }
    fun toggle(name: String) {
        val next = if (name in chosen) chosen - name else if (chosen.size < max) chosen + name else chosen
        chosen = next
        onChange(next)
    }
    TipsarrSheet(onDismiss, fullHeight = true) {
        Column(Modifier.navigationBarsPadding().padding(horizontal = Tokens.Spacing.xl).padding(bottom = Tokens.Spacing.xl), verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.md)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text(
                    stringResource(R.string.m_library_genres_pick), fontSize = Tokens.FontSize.title3, fontWeight = FontWeight.SemiBold,
                    color = Tokens.palette.fg, modifier = Modifier.weight(1f).semantics { heading() },
                )
                IconButton({ chosen = emptySet(); onChange(emptySet()) }, enabled = chosen.isNotEmpty()) {
                    Icon(Icons.Outlined.Refresh, stringResource(R.string.library_reset), tint = if (chosen.isNotEmpty()) Tokens.palette.fg else Tokens.palette.mutedFg)
                }
            }
            // Always shown; it only matters once two genres are chosen.
            val modes = listOf(false to R.string.m_library_genre_all, true to R.string.m_library_genre_any)
            SingleChoiceSegmentedButtonRow(Modifier.fillMaxWidth()) {
                modes.forEachIndexed { index, (any, label) ->
                    SegmentedButton(
                        selected = anyGenre == any, onClick = { onAnyChange(any) }, enabled = chosen.size > 1,
                        shape = SegmentedButtonDefaults.itemShape(index, modes.size),
                    ) { Text(stringResource(label)) }
                }
            }
            OutlinedTextField(
                value = query, onValueChange = { query = it }, singleLine = true,
                modifier = Modifier.fillMaxWidth().defaultMinSize(minHeight = Tokens.Size.touchTarget),
                placeholder = { Text(stringResource(R.string.m_library_genres_search), color = Tokens.palette.mutedFg) },
                leadingIcon = { Icon(Icons.Outlined.Search, null, tint = Tokens.palette.mutedFg) },
                shape = RoundedCornerShape(Tokens.Radius.md),
                colors = OutlinedTextFieldDefaults.colors(
                    focusedContainerColor = Tokens.palette.muted, unfocusedContainerColor = Tokens.palette.muted,
                    focusedBorderColor = Tokens.palette.ring, unfocusedBorderColor = Tokens.palette.muted,
                    focusedTextColor = Tokens.palette.fg, unfocusedTextColor = Tokens.palette.fg, cursorColor = Tokens.palette.fg,
                ),
            )
            LazyColumn(Modifier.weight(1f, fill = false)) {
                items(shown, key = { it }) { name ->
                    val on = name in chosen
                    Row(
                        Modifier.fillMaxWidth().defaultMinSize(minHeight = Tokens.Size.touchTarget).clickable(role = Role.Checkbox) { toggle(name) },
                        horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm), verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Checkbox(on, null, colors = CheckboxDefaults.colors(checkedColor = Tokens.palette.primary, checkmarkColor = Tokens.palette.primaryFg, uncheckedColor = Tokens.palette.mutedFg))
                        Text(name, fontSize = Tokens.FontSize.body, color = Tokens.palette.fg)
                    }
                }
            }
            TipsarrButton(stringResource(R.string.m_library_apply), onDismiss, fullWidth = true)
        }
    }
}
