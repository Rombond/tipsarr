package com.brebond.tipsarr.core.support

import androidx.compose.runtime.compositionLocalOf
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateMapOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import com.brebond.tipsarr.core.api.MediaType
import com.brebond.tipsarr.core.api.RatingSource
import com.brebond.tipsarr.core.api.TipsarrApi
import com.brebond.tipsarr.core.api.movieScores
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import java.util.Locale

/**
 * Scores for posters, following the "Score on posters" preference. TMDB comes with every list; IMDb, Metacritic
 * and Rotten Tomatoes are fetched in batches for movies (shows only have TMDB scores, so they keep showing it).
 */
class RatingProvider(private val api: TipsarrApi, source: RatingSource, private val scope: CoroutineScope) {
    var source by mutableStateOf(source)
        private set
    private val scores = mutableStateMapOf<Int, Double>()
    private val asked = HashSet<Int>()
    private val pending = LinkedHashSet<Int>()
    private var flush: Job? = null

    fun changeSource(new: RatingSource) {
        if (new == source) return
        source = new
        scores.clear()
        asked.clear()
        pending.clear()
        flush?.cancel()
    }

    class Reading(val source: RatingSource, val text: String)

    /** The score to show: the chosen source for movies, TMDB for shows. Null while unknown. */
    fun reading(type: MediaType, tmdbId: Int, tmdb: Double?): Reading? {
        val shown = if (source == RatingSource.Tmdb || type == MediaType.Tv) RatingSource.Tmdb else source
        val value = if (shown == RatingSource.Tmdb) tmdb else scores[tmdbId]
        return format(value, shown)?.let { Reading(shown, it) }
    }

    /** Asks for a movie's score; requests are grouped so one screen makes one or two calls. */
    fun request(type: MediaType, tmdbId: Int) {
        if (source == RatingSource.Tmdb || type != MediaType.Movie || !asked.add(tmdbId)) return
        pending.add(tmdbId)
        flush?.cancel()
        flush = scope.launch {
            delay(200)
            send()
        }
    }

    private suspend fun send() {
        val current = source
        val ids = pending.toList()
        pending.clear()
        // The `ids` parameter is limited to 400 characters: about 40 ids per call.
        for (chunk in ids.chunked(40)) {
            val found = runCatching { api.movieScores(chunk, current) }.getOrNull() ?: continue
            if (current != source) return
            for ((id, summary) in found) {
                val value = when (current) {
                    RatingSource.Imdb -> summary.imdb?.value
                    RatingSource.Metacritic -> summary.metacritic?.value
                    RatingSource.RottenTomatoes -> summary.rottenTomatoes?.value
                    RatingSource.Tmdb -> null
                }
                if (value != null) scores[id] = value
            }
        }
    }

    companion object {
        fun format(value: Double?, source: RatingSource): String? {
            if (value == null || value <= 0) return null
            return when (source) {
                RatingSource.Tmdb, RatingSource.Imdb -> String.format(Locale.getDefault(), "%.1f", value)
                RatingSource.Metacritic -> value.toInt().toString()
                RatingSource.RottenTomatoes -> "${value.toInt()}%"
            }
        }

        fun label(source: RatingSource): String = when (source) {
            RatingSource.Tmdb -> "TMDB"
            RatingSource.Imdb -> "IMDb"
            RatingSource.Metacritic -> "MC"
            RatingSource.RottenTomatoes -> "RT"
        }
    }
}

val LocalRatings = compositionLocalOf<RatingProvider?> { null }
