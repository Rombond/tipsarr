package com.brebond.tipsarr.ui

import androidx.compose.foundation.ExperimentalFoundationApi
import androidx.compose.foundation.clickable
import androidx.compose.foundation.combinedClickable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.size
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.Add
import androidx.compose.material.icons.outlined.BookmarkBorder
import androidx.compose.material.icons.outlined.VisibilityOff
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import com.brebond.tipsarr.R
import com.brebond.tipsarr.core.api.ApiError
import com.brebond.tipsarr.core.support.LocalTitleMenu
import com.brebond.tipsarr.core.support.LocalToast
import com.brebond.tipsarr.core.support.TitleMenu
import com.brebond.tipsarr.core.support.ToastKind
import kotlinx.coroutines.launch
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.brebond.tipsarr.core.api.MediaItem
import com.brebond.tipsarr.core.api.RatingSource
import com.brebond.tipsarr.core.images.RemoteImage
import com.brebond.tipsarr.core.images.TmdbSize
import com.brebond.tipsarr.design.Tokens
import com.brebond.tipsarr.design.palette
import java.util.Locale
import androidx.compose.ui.unit.sp
import com.brebond.tipsarr.core.support.LocalRatings
import com.brebond.tipsarr.core.support.RatingProvider

/** Standard card for a list item: poster from the server, year, score and request state; a tap opens the title. */
@OptIn(ExperimentalFoundationApi::class)
@Composable
fun MediaPoster(item: MediaItem, onClick: () -> Unit, modifier: Modifier = Modifier, longPressMenu: Boolean = true, subtitle: String? = null, rank: Int? = null) {
    val menu = if (longPressMenu) LocalTitleMenu.current else null
    val toast = LocalToast.current
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    var open by remember { mutableStateOf(false) }
    fun run(done: Int, work: suspend TitleMenu.() -> Unit) {
        open = false
        if (menu == null) return
        scope.launch {
            try { menu.work(); toast?.show(context.getString(done)) } catch (e: ApiError) { toast?.show(e.message(context), ToastKind.Error) }
        }
    }
    Box {
        PosterCard(
            title = item.title,
            modifier = if (menu == null) modifier.clickable(onClick = onClick) else modifier.combinedClickable(onClick = onClick, onLongClick = { open = true }),
            subtitle = subtitle ?: item.releaseYear,
            rank = rank,
            state = item.state,
            rating = { ScoreLabel(item.type, item.tmdbId, item.voteAverage) },
            poster = { RemoteImage(item.posterPath, TmdbSize.W342, Modifier.fillMaxSize()) },
        )
        DropdownMenu(open, { open = false }, containerColor = Tokens.palette.card) {
            if (item.state == null) PosterMenuItem(Icons.Outlined.Add, R.string.media_request) { run(R.string.m_menu_requested) { request(item.type, item.tmdbId) } }
            PosterMenuItem(Icons.Outlined.BookmarkBorder, R.string.actions_watchlist) { run(R.string.m_menu_watchlisted) { watchlist(item.type, item.tmdbId) } }
            PosterMenuItem(Icons.Outlined.VisibilityOff, R.string.media_not_interested) { run(R.string.m_menu_hidden) { hide(item.type, item.tmdbId) } }
        }
    }
}

@Composable
private fun PosterMenuItem(icon: ImageVector, text: Int, onClick: () -> Unit) {
    DropdownMenuItem(text = { Text(stringResource(text), color = Tokens.palette.fg) }, leadingIcon = { Icon(icon, null, tint = Tokens.palette.fg) }, onClick = onClick)
}

/** Provider logo and score, e.g. the TMDB logo and "7.8". */
@Composable
fun ScoreLabel(score: Double, modifier: Modifier = Modifier) {
    Row(modifier, horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.xs), verticalAlignment = Alignment.CenterVertically) {
        ProviderMark(RatingSource.Tmdb, height = 12.dp)
        Text(String.format(Locale.getDefault(), "%.1f", score), fontSize = Tokens.FontSize.footnote, lineHeight = Tokens.FontSize.footnote * 1.25f, fontWeight = FontWeight.Medium, color = Tokens.palette.mutedFg)
    }
}

/** Score for a poster following the "Score on posters" preference: the provider's logo and value (TMDB for shows). */
@Composable
fun ScoreLabel(type: com.brebond.tipsarr.core.api.MediaType, tmdbId: Int, tmdb: Double?, modifier: Modifier = Modifier) {
    val provider = LocalRatings.current
    androidx.compose.runtime.LaunchedEffect(type, tmdbId, provider?.source) { provider?.request(type, tmdbId) }
    val reading = provider?.reading(type, tmdbId, tmdb)
    when {
        reading == null -> tmdb?.takeIf { it > 0 }?.let { ScoreLabel(it, modifier) }
        reading.source == RatingSource.Metacritic -> MetacriticSquare(reading.text.toDoubleOrNull(), modifier, height = 12.dp)
        else -> Row(modifier, horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.xs), verticalAlignment = Alignment.CenterVertically) {
            ProviderMark(reading.source, value = reading.text.removeSuffix("%").replace(',', '.').toDoubleOrNull(), height = 12.dp)
            Text(reading.text, fontSize = Tokens.FontSize.footnote, lineHeight = Tokens.FontSize.footnote * 1.25f, fontWeight = FontWeight.Medium, color = Tokens.palette.mutedFg)
        }
    }
}
