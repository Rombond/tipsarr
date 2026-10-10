package com.brebond.tipsarr.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.defaultMinSize
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import coil3.compose.AsyncImage
import coil3.request.ImageRequest
import com.brebond.tipsarr.core.api.RatingSource
import com.brebond.tipsarr.core.images.LocalAppImageLoader
import com.brebond.tipsarr.design.Tokens

/**
 * Logo of a score provider (TMDB, IMDb, Rotten Tomatoes) or Metacritic's coloured square, as on iOS.
 * Logos come from Seerr (MIT), see `android/RATING-LOGOS-NOTICE.md`.
 */
@Composable
fun ProviderMark(source: RatingSource, modifier: Modifier = Modifier, value: Double? = null, height: Dp = 14.dp) {
    when (source) {
        // The logo has white letters: it needs TMDB's dark blue behind it.
        RatingSource.Tmdb -> Logo(
            "tmdb", 190.24f / 81.52f, height * 0.6f,
            modifier.clip(RoundedCornerShape(height * 0.25f)).background(Color(0xFF0D253F)).padding(horizontal = height * 0.3f, vertical = height * 0.22f),
        )
        RatingSource.Imdb -> Logo("imdb", 575f / 289.83f, height, modifier)
        RatingSource.RottenTomatoes -> Logo(if ((value ?: 100.0) >= 60) "rt_fresh" else "rt_rotten", 1f, height * 1.15f, modifier)
        RatingSource.Metacritic -> MetacriticSquare(value, modifier, height = height)
    }
}

@Composable
private fun Logo(name: String, ratio: Float, height: Dp, modifier: Modifier) {
    val context = LocalContext.current
    val loader = LocalAppImageLoader.current ?: return
    AsyncImage(
        model = ImageRequest.Builder(context).data("file:///android_asset/ratings/$name.svg").build(),
        imageLoader = loader,
        contentDescription = null,
        modifier = modifier.height(height).aspectRatio(ratio),
    )
}

/** Metacritic's score in its own colour bands (green from 61, amber from 40, red below). */
@Composable
fun MetacriticSquare(value: Double?, modifier: Modifier = Modifier, label: String? = null, height: Dp = 14.dp) {
    val color = when {
        value == null -> Tokens.Status.requested
        value >= 61 -> Tokens.Status.available
        value >= 40 -> Tokens.Status.requested
        else -> Tokens.Status.failed
    }
    Row(
        modifier.defaultMinSize(minWidth = height * 1.4f, minHeight = height * 1.4f).clip(RoundedCornerShape(height * 0.25f)).background(color).padding(horizontal = height * 0.3f),
        horizontalArrangement = Arrangement.Center, verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(label ?: value?.let { Math.round(it).toString() }.orEmpty(), fontSize = (height.value * 0.72f).sp, fontWeight = FontWeight.ExtraBold, color = Color.White)
    }
}
