package com.brebond.tipsarr.features.search

import android.content.Context
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import com.brebond.tipsarr.core.api.ApiError
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
    var movieGenres by mutableStateOf<List<Genre>>(emptyList())
        private set
    var tvGenres by mutableStateOf<List<Genre>>(emptyList())
        private set
    var recents by mutableStateOf(prefs.getString(RECENTS, null)?.split('\n')?.filter { it.isNotEmpty() } ?: emptyList())
        private set

    private var page = 0
    private var hasMore = false
    private var currentQuery = ""

    val trimmed: String get() = query.trim()

    /** Titles for the selected scope (people have their own section). */
    val visibleTitles: List<MediaItem>
        get() = when (scope) {
            SearchScope.All -> items
            SearchScope.Movies -> items.filter { it.type == MediaType.Movie }
            SearchScope.Tv -> items.filter { it.type == MediaType.Tv }
            SearchScope.People -> emptyList()
        }

    val hasResults: Boolean
        get() = when (scope) {
            SearchScope.All -> items.isNotEmpty() || people.isNotEmpty()
            SearchScope.People -> people.isNotEmpty()
            else -> visibleTitles.isNotEmpty()
        }

    suspend fun loadGenres() {
        if (movieGenres.isNotEmpty()) return
        coroutineScope {
            val movies = async { runCatching { api.genres(MediaType.Movie) }.getOrNull() }
            val shows = async { runCatching { api.genres(MediaType.Tv) }.getOrNull() }
            movieGenres = movies.await().orEmpty()
            tvGenres = shows.await().orEmpty()
        }
    }

    fun remember() {
        val text = trimmed
        if (text.isEmpty()) return
        recents = (listOf(text) + recents.filterNot { it.equals(text, ignoreCase = true) }).take(8)
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
