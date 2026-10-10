package com.brebond.tipsarr.ui

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.size
import androidx.compose.material3.Text
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
