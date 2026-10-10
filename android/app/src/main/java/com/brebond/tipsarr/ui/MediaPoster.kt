package com.brebond.tipsarr.ui

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.size
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Star
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.brebond.tipsarr.core.api.MediaItem
import com.brebond.tipsarr.core.images.RemoteImage
import com.brebond.tipsarr.core.images.TmdbSize
import com.brebond.tipsarr.design.Tokens
import com.brebond.tipsarr.design.palette
import java.util.Locale
import androidx.compose.ui.unit.sp
import com.brebond.tipsarr.core.support.LocalRatings
import com.brebond.tipsarr.core.support.RatingProvider

/** Standard card for a list item: poster from the server, year, score and request state; a tap opens the title. */
@Composable
fun MediaPoster(item: MediaItem, onClick: () -> Unit, modifier: Modifier = Modifier) {
    PosterCard(
        title = item.title,
        modifier = modifier.clickable(onClick = onClick),
        subtitle = item.releaseYear,
        state = item.state,
        rating = { ScoreLabel(item.type, item.tmdbId, item.voteAverage) },
        poster = { RemoteImage(item.posterPath, TmdbSize.W342, Modifier.fillMaxSize()) },
    )
}

/** TMDB score with a star (the provider logos of the "score on posters" setting come with Settings). */
@Composable
fun ScoreLabel(score: Double, modifier: Modifier = Modifier) {
    Row(modifier, horizontalArrangement = Arrangement.spacedBy(2.dp), verticalAlignment = Alignment.CenterVertically) {
        Icon(Icons.Filled.Star, contentDescription = null, tint = Tokens.Status.requested, modifier = Modifier.size(12.dp))
        Text(String.format(Locale.getDefault(), "%.1f", score), fontSize = Tokens.FontSize.footnote, lineHeight = Tokens.FontSize.footnote * 1.25f, fontWeight = FontWeight.Medium, color = Tokens.palette.mutedFg)
    }
}

/** Score for a poster following the "Score on posters" preference: TMDB with a star, or the chosen provider's name and value. */
@Composable
fun ScoreLabel(type: com.brebond.tipsarr.core.api.MediaType, tmdbId: Int, tmdb: Double?, modifier: Modifier = Modifier) {
    val provider = LocalRatings.current
    androidx.compose.runtime.LaunchedEffect(type, tmdbId, provider?.source) { provider?.request(type, tmdbId) }
    val reading = provider?.reading(type, tmdbId, tmdb)
    when {
        reading == null -> tmdb?.takeIf { it > 0 }?.let { ScoreLabel(it, modifier) }
        reading.source == com.brebond.tipsarr.core.api.RatingSource.Tmdb -> ScoreLabel(tmdb ?: 0.0, modifier)
        else -> Row(modifier, horizontalArrangement = Arrangement.spacedBy(3.dp), verticalAlignment = Alignment.CenterVertically) {
            Text(RatingProvider.label(reading.source), fontSize = 10.sp, fontWeight = FontWeight.Bold, color = Tokens.palette.mutedFg)
            Text(reading.text, fontSize = Tokens.FontSize.footnote, lineHeight = Tokens.FontSize.footnote * 1.25f, fontWeight = FontWeight.Medium, color = Tokens.palette.mutedFg)
        }
    }
}
