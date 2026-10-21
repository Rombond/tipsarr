package com.brebond.tipsarr.features.search

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.defaultMinSize
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.GridItemSpan
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.itemsIndexed
import androidx.compose.foundation.lazy.grid.items as gridItems
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.outlined.KeyboardArrowRight
import androidx.compose.material.icons.outlined.Close
import androidx.compose.material.icons.outlined.Delete
import androidx.compose.material.icons.outlined.FilterList
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.SegmentedButton
import androidx.compose.material3.SegmentedButtonDefaults
import androidx.compose.material3.SingleChoiceSegmentedButtonRow
import androidx.compose.material3.SwipeToDismissBox
import androidx.compose.material3.SwipeToDismissBoxValue
import androidx.compose.material3.rememberSwipeToDismissBoxState
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.material.icons.outlined.History
import androidx.compose.material.icons.outlined.Person
import androidx.compose.material.icons.outlined.Search
import androidx.compose.material.icons.outlined.WarningAmber
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.OutlinedTextFieldDefaults
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.LocalFocusManager
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.heading
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.brebond.tipsarr.R
import com.brebond.tipsarr.core.api.ApiError
import com.brebond.tipsarr.core.api.MediaItem
import com.brebond.tipsarr.core.api.PersonSummary
import com.brebond.tipsarr.core.images.RemoteImage
import com.brebond.tipsarr.core.images.TmdbSize
import com.brebond.tipsarr.design.Tokens
import com.brebond.tipsarr.design.palette
import com.brebond.tipsarr.features.discover.Phase
import com.brebond.tipsarr.features.person.PersonRoute
import com.brebond.tipsarr.ui.MediaPoster
import com.brebond.tipsarr.ui.OfflineState
import com.brebond.tipsarr.ui.PosterSkeleton
import com.brebond.tipsarr.ui.StateView
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

private val GUTTER = Tokens.Spacing.lg

@Composable
fun SearchScreen(
    model: SearchModel,
    onOpenItem: (MediaItem) -> Unit,
    onOpenPerson: (PersonRoute) -> Unit,
    onOpenGenre: (MergedGenre) -> Unit,
    modifier: Modifier = Modifier,
) {
    // Debounce: typing restarts this effect before the request starts.
    LaunchedEffect(model.query) {
        delay(350)
        model.search()
    }
    Column(modifier.fillMaxSize()) {
        Column(Modifier.padding(horizontal = GUTTER).padding(top = Tokens.Spacing.sm), verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.md)) {
            // Inline like the iOS navigation bar: the title never grows into a large one.
            Text(
                stringResource(R.string.m_tab_search), fontSize = Tokens.FontSize.headline, fontWeight = FontWeight.SemiBold,
                color = Tokens.palette.fg, textAlign = TextAlign.Center, modifier = Modifier.fillMaxWidth().semantics { heading() },
            )
            SearchField(model)
            if (model.trimmed.isNotEmpty()) {
                var filtersOpen by remember { mutableStateOf(false) }
                Row(horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.md), verticalAlignment = Alignment.CenterVertically) {
                    val scopes = SearchScope.entries
                    SingleChoiceSegmentedButtonRow(Modifier.weight(1f)) {
                        scopes.forEachIndexed { index, scope ->
                            SegmentedButton(
                                selected = model.scope == scope, onClick = { model.scope = scope },
                                shape = SegmentedButtonDefaults.itemShape(index, scopes.size),
                                contentPadding = PaddingValues(horizontal = Tokens.Spacing.xs),
                            ) { Text(stringResource(scope.title()), maxLines = 1, overflow = TextOverflow.Ellipsis) }
                        }
                    }
                    if (model.scope != SearchScope.People) FiltersButton(model.filters.activeCount) { filtersOpen = true }
                }
                if (filtersOpen) SearchFiltersSheet(model) { filtersOpen = false }
            }
        }
        Box(Modifier.weight(1f).fillMaxWidth()) {
            if (model.trimmed.isEmpty()) Landing(model, onOpenGenre) else Results(model, onOpenItem, onOpenPerson)
        }
    }
}

/** Round button next to the scopes (the iOS glass circle); filled with the accent while a filter is active. */
@Composable
private fun FiltersButton(activeCount: Int, onClick: () -> Unit) {
    val active = activeCount > 0
    Box(
        Modifier.size(Tokens.Size.touchTarget).clip(CircleShape).background(if (active) Tokens.palette.primary else Tokens.palette.muted)
            .clickable(role = Role.Button, onClick = onClick),
        contentAlignment = Alignment.Center,
    ) {
        Icon(
            Icons.Outlined.FilterList, stringResource(R.string.m_search_filters),
            tint = if (active) Tokens.palette.primaryFg else Tokens.palette.fg,
        )
    }
}

private fun SearchScope.title(): Int = when (this) {
    SearchScope.All -> R.string.search_tab_all
    SearchScope.Movies -> R.string.type_movies
    SearchScope.Tv -> R.string.type_shows
    SearchScope.People -> R.string.type_people
}

@Composable
private fun SearchField(model: SearchModel) {
    val focus = LocalFocusManager.current
    OutlinedTextField(
        value = model.query,
        onValueChange = { model.query = it },
        modifier = Modifier.fillMaxWidth().defaultMinSize(minHeight = Tokens.Size.touchTarget),
        placeholder = { Text(stringResource(R.string.m_search_prompt), color = Tokens.palette.mutedFg) },
        leadingIcon = { Icon(Icons.Outlined.Search, null, tint = Tokens.palette.mutedFg) },
        trailingIcon = {
            if (model.query.isNotEmpty()) IconButton({ model.query = "" }) { Icon(Icons.Outlined.Close, stringResource(R.string.common_dismiss), tint = Tokens.palette.mutedFg) }
        },
        singleLine = true,
        keyboardOptions = KeyboardOptions(imeAction = ImeAction.Search),
        keyboardActions = KeyboardActions(onSearch = { model.remember(); focus.clearFocus() }),
        shape = CircleShape,
        colors = OutlinedTextFieldDefaults.colors(
            focusedContainerColor = Tokens.palette.muted, unfocusedContainerColor = Tokens.palette.muted,
            focusedBorderColor = Tokens.palette.ring, unfocusedBorderColor = Tokens.palette.muted,
            focusedTextColor = Tokens.palette.fg, unfocusedTextColor = Tokens.palette.fg, cursorColor = Tokens.palette.fg,
        ),
    )
}

// Landing (no query): recent searches, then the genres.

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun Landing(model: SearchModel, onOpenGenre: (MergedGenre) -> Unit) {
    LaunchedEffect(Unit) { model.loadGenres() }
    LazyColumn(Modifier.fillMaxSize()) {
        if (model.recents.isNotEmpty()) {
            item(key = "recents-header") {
                Row(Modifier.fillMaxWidth().padding(horizontal = GUTTER).defaultMinSize(minHeight = Tokens.Size.touchTarget), verticalAlignment = Alignment.CenterVertically) {
                    SectionTitle(stringResource(R.string.m_search_recents), Modifier.weight(1f))
                    TextButton(model::clearRecents) { Text(stringResource(R.string.m_search_clear)) }
                }
            }
            // Swipe a recent search to the left to delete it.
            items(model.recents, key = { "recent-$it" }) { recent ->
                val swipe = rememberSwipeToDismissBoxState(confirmValueChange = { value ->
                    if (value == SwipeToDismissBoxValue.EndToStart) { model.removeRecent(recent); true } else false
                })
                SwipeToDismissBox(
                    swipe, enableDismissFromStartToEnd = false,
                    backgroundContent = {
                        Box(Modifier.fillMaxSize().background(Tokens.palette.destructive).padding(horizontal = GUTTER), contentAlignment = Alignment.CenterEnd) {
                            Icon(Icons.Outlined.Delete, stringResource(R.string.m_search_clear), tint = Color.White)
                        }
                    },
                ) {
                    Row(
                        Modifier.fillMaxWidth().background(Tokens.palette.bg).defaultMinSize(minHeight = Tokens.Size.touchTarget)
                            .clickable { model.query = recent }.padding(horizontal = GUTTER),
                        horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.md), verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Icon(Icons.Outlined.History, null, tint = Tokens.palette.fg)
                        Text(recent, fontSize = Tokens.FontSize.body, color = Tokens.palette.fg, maxLines = 1, overflow = TextOverflow.Ellipsis)
                    }
                }
            }
        }
        if (model.genres.isNotEmpty()) item(key = "genres") {
            // The chips sit at the left edge whether or not the recents are shown.
            Column(Modifier.fillMaxWidth().padding(horizontal = GUTTER).padding(top = Tokens.Spacing.x2xl, bottom = GUTTER), verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.lg)) {
                SectionTitle(stringResource(R.string.m_search_genres))
                GenreChips(model.genres, onOpenGenre)
            }
        }
    }
}

@Composable
private fun SectionTitle(text: String, modifier: Modifier = Modifier) {
    Text(text, fontSize = Tokens.FontSize.title3, fontWeight = FontWeight.SemiBold, color = Tokens.palette.fg, modifier = modifier.semantics { heading() })
}

@OptIn(ExperimentalLayoutApi::class)
@Composable
private fun GenreChips(genres: List<MergedGenre>, onClick: (MergedGenre) -> Unit) {
    FlowRow(horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm), verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm)) {
        genres.forEach { genre ->
            Text(
                genre.name, fontSize = Tokens.FontSize.subhead, fontWeight = FontWeight.Medium, color = Tokens.palette.fg,
                modifier = Modifier.defaultMinSize(minHeight = Tokens.Size.touchTarget - Tokens.Spacing.sm).clip(CircleShape)
                    .background(Tokens.palette.muted).clickable { onClick(genre) }.padding(horizontal = Tokens.Spacing.md)
                    .wrapContentHeightCentered(),
            )
        }
    }
}

private fun Modifier.wrapContentHeightCentered(): Modifier = this.then(Modifier.padding(vertical = 10.dp))

// Results

@Composable
private fun Results(model: SearchModel, onOpenItem: (MediaItem) -> Unit, onOpenPerson: (PersonRoute) -> Unit) {
    val scope = rememberCoroutineScope()
    val context = LocalContext.current
    when (val phase = model.phase) {
        Phase.Idle, Phase.Loading -> LazyVerticalGrid(
            GridCells.Adaptive(104.dp), Modifier.fillMaxSize(), contentPadding = PaddingValues(GUTTER),
            verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.lg), horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.md),
        ) { gridItems(List(9) { it }) { PosterSkeleton() } }
        is Phase.Failed -> Box(Modifier.fillMaxSize()) {
            if (phase.error == ApiError.Unreachable) OfflineState({ scope.launch { model.search() } })
            else StateView(Icons.Outlined.WarningAmber, phase.error.message(context), actionTitle = stringResource(R.string.common_retry), onAction = { scope.launch { model.search() } })
        }
        Phase.Loaded -> if (!model.hasResults) {
            if (model.filters.activeCount > 0 && model.scope != SearchScope.People && model.items.isNotEmpty()) {
                StateView(Icons.Outlined.Search, stringResource(R.string.common_no_results))
            } else {
                StateView(Icons.Outlined.Search, stringResource(R.string.search_nothing, model.trimmed), message = stringResource(R.string.search_nothing_hint))
            }
        } else {
            val titles = model.visibleTitles
            LazyVerticalGrid(
                GridCells.Adaptive(104.dp), Modifier.fillMaxSize(), contentPadding = PaddingValues(GUTTER),
                verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.lg), horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.md),
            ) {
                if (model.scope == SearchScope.All || model.scope == SearchScope.People) {
                    if (model.people.isNotEmpty()) {
                        item(span = { GridItemSpan(maxLineSpan) }) { SectionTitle(stringResource(R.string.type_people)) }
                        if (model.scope == SearchScope.People) {
                            gridItems(model.people, key = { "p${it.id}" }, span = { GridItemSpan(maxLineSpan) }) { person ->
                                PersonRow(person) { model.remember(); onOpenPerson(PersonRoute(person.id, person.name)) }
                            }
                        } else item(span = { GridItemSpan(maxLineSpan) }) {
                            LazyRow(horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.md)) {
                                items(model.people.take(12), key = { it.id }) { person -> PersonBubble(person) { onOpenPerson(PersonRoute(person.id, person.name)) } }
                            }
                        }
                    }
                }
                if (model.scope != SearchScope.People) {
                    itemsIndexed(titles, key = { _, item -> item.id }) { index, item ->
                        LaunchedEffect(index, titles.size) { model.loadMore(index) }
                        MediaPoster(item, { model.remember(); onOpenItem(item) })
                    }
                    if (model.loadingMore) item(span = { GridItemSpan(maxLineSpan) }) {
                        Box(Modifier.fillMaxWidth().padding(Tokens.Spacing.lg), contentAlignment = Alignment.Center) { CircularProgressIndicator() }
                    }
                }
            }
        }
    }
}

@Composable
fun PersonPhoto(path: String?, size: androidx.compose.ui.unit.Dp, modifier: Modifier = Modifier) {
    Box(modifier.size(size).clip(CircleShape).background(Tokens.palette.muted), contentAlignment = Alignment.Center) {
        Icon(Icons.Outlined.Person, null, tint = Tokens.palette.mutedFg)
        RemoteImage(path, TmdbSize.W185, Modifier.fillMaxSize())
    }
}

@Composable
private fun PersonBubble(person: PersonSummary, onClick: () -> Unit) {
    Column(Modifier.width(88.dp).clickable(onClick = onClick), horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.xs)) {
        PersonPhoto(person.profilePath, 72.dp)
        Text(person.name, fontSize = Tokens.FontSize.caption, fontWeight = FontWeight.Medium, color = Tokens.palette.fg, maxLines = 1, overflow = TextOverflow.Ellipsis)
        person.department?.takeIf { it.isNotEmpty() }?.let { Text(it, fontSize = 11.sp, color = Tokens.palette.mutedFg, maxLines = 1, overflow = TextOverflow.Ellipsis) }
    }
}

@Composable
private fun PersonRow(person: PersonSummary, onClick: () -> Unit) {
    Row(
        Modifier.fillMaxWidth().defaultMinSize(minHeight = Tokens.Size.touchTarget + Tokens.Spacing.md).clickable(onClick = onClick),
        horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.md), verticalAlignment = Alignment.CenterVertically,
    ) {
        PersonPhoto(person.profilePath, 52.dp)
        Column(Modifier.weight(1f)) {
            Text(person.name, fontSize = Tokens.FontSize.body, fontWeight = FontWeight.Medium, color = Tokens.palette.fg)
            person.department?.takeIf { it.isNotEmpty() }?.let { Text(it, fontSize = Tokens.FontSize.footnote, color = Tokens.palette.mutedFg) }
        }
        Icon(Icons.AutoMirrored.Outlined.KeyboardArrowRight, null, tint = Tokens.palette.mutedFg)
    }
}
