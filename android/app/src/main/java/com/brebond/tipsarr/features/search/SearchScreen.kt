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
import com.brebond.tipsarr.core.api.Genre
import com.brebond.tipsarr.core.api.MediaItem
import com.brebond.tipsarr.core.api.MediaType
import com.brebond.tipsarr.core.api.PersonSummary
import com.brebond.tipsarr.core.images.RemoteImage
import com.brebond.tipsarr.core.images.TmdbSize
import com.brebond.tipsarr.design.Tokens
import com.brebond.tipsarr.design.palette
import com.brebond.tipsarr.features.discover.Phase
import com.brebond.tipsarr.features.person.PersonRoute
import com.brebond.tipsarr.ui.Chip
import com.brebond.tipsarr.ui.MediaPoster
import com.brebond.tipsarr.ui.OfflineState
import com.brebond.tipsarr.ui.PosterSkeleton
import com.brebond.tipsarr.ui.StateView
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

private val GUTTER = Tokens.Spacing.lg

class GenreOpen(val type: MediaType, val genre: Genre)

@Composable
fun SearchScreen(
    model: SearchModel,
    onOpenItem: (MediaItem) -> Unit,
    onOpenPerson: (PersonRoute) -> Unit,
    onOpenGenre: (MediaType, Genre) -> Unit,
    modifier: Modifier = Modifier,
) {
    // Debounce: typing restarts this effect before the request starts.
    LaunchedEffect(model.query) {
        delay(350)
        model.search()
    }
    Column(modifier.fillMaxSize()) {
        Column(Modifier.padding(horizontal = GUTTER).padding(top = Tokens.Spacing.sm), verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.md)) {
            Text(
                stringResource(R.string.m_tab_search), fontSize = Tokens.FontSize.largeTitle, fontWeight = FontWeight.Bold,
                color = Tokens.palette.fg, modifier = Modifier.semantics { heading() },
            )
            SearchField(model)
            if (model.trimmed.isNotEmpty()) {
                LazyRow(horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm)) {
                    items(SearchScope.entries) { scope ->
                        Chip(stringResource(scope.title()), selected = model.scope == scope, onClick = { model.scope = scope })
                    }
                }
            }
        }
        Box(Modifier.weight(1f).fillMaxWidth()) {
            if (model.trimmed.isEmpty()) Landing(model, onOpenGenre) else Results(model, onOpenItem, onOpenPerson)
        }
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

@OptIn(ExperimentalLayoutApi::class)
@Composable
private fun Landing(model: SearchModel, onOpenGenre: (MediaType, Genre) -> Unit) {
    LaunchedEffect(Unit) { model.loadGenres() }
    LazyColumn(Modifier.fillMaxSize(), contentPadding = PaddingValues(GUTTER), verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.x2xl)) {
        if (model.recents.isNotEmpty()) item {
            Column(verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm)) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    SectionTitle(stringResource(R.string.m_search_recents), Modifier.weight(1f))
                    TextButton(model::clearRecents) { Text(stringResource(R.string.m_search_clear)) }
                }
                model.recents.forEach { recent ->
                    Row(
                        Modifier.fillMaxWidth().defaultMinSize(minHeight = Tokens.Size.touchTarget).clickable { model.query = recent },
                        horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.md), verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Icon(Icons.Outlined.History, null, tint = Tokens.palette.fg)
                        Text(recent, fontSize = Tokens.FontSize.body, color = Tokens.palette.fg)
                    }
                }
            }
        }
        if (model.movieGenres.isNotEmpty() || model.tvGenres.isNotEmpty()) item {
            Column(verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.lg)) {
                SectionTitle(stringResource(R.string.m_search_genres))
                GenreGroup(stringResource(R.string.type_movies), model.movieGenres) { onOpenGenre(MediaType.Movie, it) }
                GenreGroup(stringResource(R.string.type_shows), model.tvGenres) { onOpenGenre(MediaType.Tv, it) }
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
private fun GenreGroup(title: String, genres: List<Genre>, onClick: (Genre) -> Unit) {
    Column(verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm)) {
        Text(title, fontSize = Tokens.FontSize.subhead, fontWeight = FontWeight.Medium, color = Tokens.palette.mutedFg)
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
            StateView(Icons.Outlined.Search, stringResource(R.string.search_nothing, model.trimmed), message = stringResource(R.string.search_nothing_hint))
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
