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

/** Standard card for a list item: poster from the server, year, score and request state; a tap opens the title. */
@Composable
fun MediaPoster(item: MediaItem, onClick: () -> Unit, modifier: Modifier = Modifier) {
    PosterCard(
        title = item.title,
        modifier = modifier.clickable(onClick = onClick),
        subtitle = item.releaseYear,
        state = item.state,
        rating = item.voteAverage.takeIf { it > 0 }?.let { score -> { ScoreLabel(score) } },
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
