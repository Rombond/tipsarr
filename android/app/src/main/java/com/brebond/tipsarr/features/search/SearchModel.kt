package com.brebond.tipsarr.features.search

import android.content.Context
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import com.brebond.tipsarr.core.api.ApiError
import com.brebond.tipsarr.core.api.Availability
import com.brebond.tipsarr.core.api.Genre
import com.brebond.tipsarr.core.api.MediaItem
import com.brebond.tipsarr.core.api.MediaType
import com.brebond.tipsarr.core.api.PersonSummary
import com.brebond.tipsarr.core.api.TipsarrApi
import com.brebond.tipsarr.core.api.genres
import com.brebond.tipsarr.core.api.search
import com.brebond.tipsarr.features.discover.Phase
import kotlinx.coroutines.async
import kotlinx.coroutines.coroutineScope

enum class SearchScope { All, Movies, Tv, People }

/** TMDB keeps one genre list for movies and another for shows; a name that exists in both is one entry here. */
data class MergedGenre(val name: String, val movieId: Int?, val tvId: Int?)

fun mergeGenres(movies: List<Genre>, shows: List<Genre>): List<MergedGenre> {
    val byName = linkedMapOf<String, MergedGenre>()
    movies.forEach { byName[it.name.lowercase()] = MergedGenre(it.name, it.id, null) }
    shows.forEach { g ->
        val key = g.name.lowercase()
        byName[key] = byName[key]?.copy(tvId = g.id) ?: MergedGenre(g.name, null, g.id)
    }
    return byName.values.sortedBy { it.name.lowercase() }
}

enum class SearchAvailability { Any, Missing, InLibrary }

enum class SearchSort { Relevance, Rating, Year }

/** What the Filters sheet on Search can change. They apply on the titles already loaded. */
data class SearchFilters(
    val genres: Set<String> = emptySet(),
    val anyGenre: Boolean = false,
    val yearFrom: Int? = null,
    val yearTo: Int? = null,
    val minRating: Double? = null,
    val availability: SearchAvailability = SearchAvailability.Any,
    val sort: SearchSort = SearchSort.Relevance,
) {
    val activeCount: Int
        get() = listOf(genres.isNotEmpty(), yearFrom != null || yearTo != null, minRating != null, availability != SearchAvailability.Any, sort != SearchSort.Relevance).count { it }
}

class SearchModel(private val api: TipsarrApi, context: Context) {
    private val prefs = context.applicationContext.getSharedPreferences("search", Context.MODE_PRIVATE)

    var query by mutableStateOf("")
    var scope by mutableStateOf(SearchScope.All)
    var phase by mutableStateOf<Phase>(Phase.Idle)
        private set
    var items by mutableStateOf<List<MediaItem>>(emptyList())
        private set
    var people by mutableStateOf<List<PersonSummary>>(emptyList())
        private set
    var loadingMore by mutableStateOf(false)
        private set
    var genres by mutableStateOf<List<MergedGenre>>(emptyList())
        private set
    var filters by mutableStateOf(SearchFilters())
    var recents by mutableStateOf(prefs.getString(RECENTS, null)?.split('\n')?.filter { it.isNotEmpty() } ?: emptyList())
        private set

    private var page = 0
    private var hasMore = false
    private var currentQuery = ""

    val trimmed: String get() = query.trim()

    /** Titles for the selected scope (people have their own section), after the filters. */
    val visibleTitles: List<MediaItem>
        get() {
            val scoped = when (scope) {
                SearchScope.All -> items
                SearchScope.Movies -> items.filter { it.type == MediaType.Movie }
                SearchScope.Tv -> items.filter { it.type == MediaType.Tv }
                SearchScope.People -> return emptyList()
            }
            return applyFilters(scoped)
        }

    private fun applyFilters(list: List<MediaItem>): List<MediaItem> {
        val f = filters
        if (f.activeCount == 0) return list
        val wanted = genres.filter { it.name in f.genres }
        val kept = list.filter { item ->
            val year = item.releaseYear?.toIntOrNull()
            (f.genres.isEmpty() || wanted.map { g -> item.genreIds.any { it == g.movieId || it == g.tvId } }.let { m -> if (f.anyGenre) m.any { it } else m.size >= f.genres.size && m.all { it } }) &&
                (f.yearFrom == null || (year != null && year >= f.yearFrom)) &&
                (f.yearTo == null || (year != null && year <= f.yearTo)) &&
                (f.minRating == null || item.voteAverage >= f.minRating) &&
                when (f.availability) {
                    SearchAvailability.Any -> true
                    SearchAvailability.Missing -> item.availability == Availability.None
                    SearchAvailability.InLibrary -> item.availability != Availability.None
                }
        }
        return when (f.sort) {
            SearchSort.Relevance -> kept
            SearchSort.Rating -> kept.sortedByDescending { it.voteAverage }
            SearchSort.Year -> kept.sortedByDescending { it.releaseDate.orEmpty() }
        }
    }

    val hasResults: Boolean
        get() = when (scope) {
            SearchScope.All -> visibleTitles.isNotEmpty() || people.isNotEmpty()
            SearchScope.People -> people.isNotEmpty()
            else -> visibleTitles.isNotEmpty()
        }

    suspend fun loadGenres() {
        if (genres.isNotEmpty()) return
        coroutineScope {
            val movies = async { runCatching { api.genres(MediaType.Movie) }.getOrNull() }
            val shows = async { runCatching { api.genres(MediaType.Tv) }.getOrNull() }
            genres = mergeGenres(movies.await().orEmpty(), shows.await().orEmpty())
        }
    }

    fun remember() {
        val text = trimmed
        if (text.isEmpty()) return
        recents = (listOf(text) + recents.filterNot { it.equals(text, ignoreCase = true) }).take(8)
        prefs.edit().putString(RECENTS, recents.joinToString("\n")).apply()
    }

    fun removeRecent(text: String) {
        recents = recents.filterNot { it == text }
        prefs.edit().putString(RECENTS, recents.joinToString("\n")).apply()
    }

    fun clearRecents() {
        recents = emptyList()
        prefs.edit().remove(RECENTS).apply()
    }

    suspend fun search() {
        val text = trimmed
        if (text.isEmpty()) {
            phase = Phase.Idle
            items = emptyList()
            people = emptyList()
            return
        }
        currentQuery = text
        if (items.isEmpty() && people.isEmpty()) phase = Phase.Loading
        try {
            val result = api.search(text, 1)
            if (currentQuery != text) return
            items = result.items.distinctBy { it.id }
            people = result.people
            page = result.page
            hasMore = result.hasMore
            phase = Phase.Loaded
        } catch (e: ApiError) {
            if (currentQuery == text) phase = Phase.Failed(e)
        }
    }

    suspend fun loadMore(index: Int) {
        if (phase != Phase.Loaded || !hasMore || loadingMore || index < items.size - 6) return
        loadingMore = true
        val text = currentQuery
        try {
            val next = api.search(text, page + 1)
            if (currentQuery != text) return
            val known = items.mapTo(HashSet()) { it.id }
            items = items + next.items.distinctBy { it.id }.filter { it.id !in known }
            page = next.page
            hasMore = next.hasMore
        } catch (e: ApiError) {
            hasMore = false
        } finally {
            loadingMore = false
        }
    }

    private companion object {
        const val RECENTS = "recents"
    }
}
