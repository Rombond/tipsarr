package com.brebond.tipsarr.features.person

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.WarningAmber
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.layout.Layout
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.heading
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.brebond.tipsarr.R
import com.brebond.tipsarr.core.api.ApiError
import com.brebond.tipsarr.core.api.MediaItem
import com.brebond.tipsarr.core.support.formatDate
import com.brebond.tipsarr.design.Tokens
import com.brebond.tipsarr.design.palette
import com.brebond.tipsarr.features.search.PersonPhoto
import com.brebond.tipsarr.ui.MediaPoster
import com.brebond.tipsarr.ui.OfflineState
import com.brebond.tipsarr.ui.StateView
import com.brebond.tipsarr.ui.TopBarScaffold
import kotlinx.coroutines.launch

/** An actor, director...: photo, biography and the titles they are known for. */
@Composable
fun PersonScreen(model: PersonModel, onBack: () -> Unit, onOpenItem: (MediaItem) -> Unit) {
    val scope = rememberCoroutineScope()
    val context = LocalContext.current
    var expanded by remember { mutableStateOf(false) }
    LaunchedEffect(model) { model.load() }
    TopBarScaffold(model.route.name, onBack) {
        val person = model.person
        val error = model.error
        when {
            person != null -> Column(
                Modifier.fillMaxSize().verticalScroll(rememberScrollState()).padding(Tokens.Spacing.lg),
                verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.xl),
            ) {
                Row(horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.lg), verticalAlignment = Alignment.CenterVertically) {
                    PersonPhoto(person.profilePath, 96.dp)
                    Column(verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.xs)) {
                        Text(person.name, fontSize = Tokens.FontSize.title2, fontWeight = FontWeight.Bold, color = Tokens.palette.fg, modifier = Modifier.semantics { heading() })
                        person.department?.takeIf { it.isNotEmpty() }?.let { Text(it, fontSize = Tokens.FontSize.subhead, color = Tokens.palette.mutedFg) }
                        person.birthday?.takeIf { it.isNotEmpty() }?.let {
                            Text(stringResource(R.string.m_person_born, formatDate(it)), fontSize = Tokens.FontSize.footnote, color = Tokens.palette.mutedFg)
                        }
                        person.deathday?.takeIf { it.isNotEmpty() }?.let {
                            Text(stringResource(R.string.m_person_died, formatDate(it)), fontSize = Tokens.FontSize.footnote, color = Tokens.palette.mutedFg)
                        }
                    }
                }
                person.biography?.takeIf { it.isNotEmpty() }?.let { biography ->
                    Column(verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm)) {
                        Text(stringResource(R.string.m_person_biography), fontSize = Tokens.FontSize.title3, fontWeight = FontWeight.Bold, color = Tokens.palette.fg, modifier = Modifier.semantics { heading() })
                        Text(
                            biography, fontSize = Tokens.FontSize.callout, lineHeight = Tokens.FontSize.callout * 1.4f, color = Tokens.palette.fg,
                            maxLines = if (expanded) Int.MAX_VALUE else 6, overflow = TextOverflow.Ellipsis,
                        )
                        TextButton({ expanded = !expanded }) { Text(stringResource(if (expanded) R.string.m_common_less else R.string.m_common_more)) }
                    }
                }
                if (person.credits.isNotEmpty()) {
                    Column(verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.md)) {
                        Text(stringResource(R.string.m_person_known_for), fontSize = Tokens.FontSize.title3, fontWeight = FontWeight.Bold, color = Tokens.palette.fg, modifier = Modifier.semantics { heading() })
                        PosterGrid(person.credits, onOpenItem)
                    }
                }
            }
            error != null -> if (error == ApiError.Unreachable) OfflineState({ scope.launch { model.load() } })
            else StateView(Icons.Outlined.WarningAmber, error.message(context), actionTitle = stringResource(R.string.common_retry), onAction = { scope.launch { model.load() } })
            else -> Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) { CircularProgressIndicator() }
        }
    }
}

/** A non-lazy adaptive grid (the screen scrolls as a whole): as many 104 dp+ columns as fit. */
@Composable
private fun PosterGrid(items: List<MediaItem>, onOpenItem: (MediaItem) -> Unit) {
    androidx.compose.foundation.layout.BoxWithConstraints(Modifier.fillMaxWidth()) {
        val gap = Tokens.Spacing.md
        val columns = maxOf(1, ((maxWidth + gap) / (104.dp + gap)).toInt())
        Column(verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.lg)) {
            items.chunked(columns).forEach { row ->
                Row(horizontalArrangement = Arrangement.spacedBy(gap)) {
                    row.forEach { item -> MediaPoster(item, { onOpenItem(item) }, Modifier.weight(1f)) }
                    repeat(columns - row.size) { Box(Modifier.weight(1f)) }
                }
            }
        }
    }
}
