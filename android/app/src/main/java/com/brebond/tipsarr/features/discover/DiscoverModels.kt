package com.brebond.tipsarr.features.discover

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import com.brebond.tipsarr.core.api.ApiError
import com.brebond.tipsarr.core.api.MediaItem
import com.brebond.tipsarr.core.api.MediaPage
import com.brebond.tipsarr.core.api.SuggestionRow
import com.brebond.tipsarr.core.api.TipsarrApi
import com.brebond.tipsarr.core.api.popularMovies
import com.brebond.tipsarr.core.api.popularTv
import com.brebond.tipsarr.core.api.suggestions
import com.brebond.tipsarr.core.api.trending
import com.brebond.tipsarr.core.api.upcoming

sealed interface Phase {
    data object Idle : Phase
    data object Loading : Phase
    data object Loaded : Phase
    data class Failed(val error: ApiError) : Phase
}

/** One paged list of titles (infinite scroll). Duplicates across pages are dropped. */
class MediaListModel(val source: Source, private val api: TipsarrApi) {
    enum class Source { Trending, Upcoming, Movies, Tv }

    var items by mutableStateOf<List<MediaItem>>(emptyList())
        private set
    var phase by mutableStateOf<Phase>(Phase.Idle)
        private set
    var loadingMore by mutableStateOf(false)
        private set
    private var page = 0
    private var hasMore = true

    suspend fun loadIfNeeded() {
        if (phase == Phase.Idle) refresh()
    }

    suspend fun refresh() {
        if (items.isEmpty()) phase = Phase.Loading
        try {
            val first = fetch(1)
            items = first.items.distinctBy { it.id }
            page = first.page
            hasMore = first.hasMore
            phase = Phase.Loaded
        } catch (e: ApiError) {
            // Keep what is on screen when a pull to refresh fails.
            if (items.isEmpty()) phase = Phase.Failed(e)
        }
    }

    /** Call from the last cells; loads the next page once. */
    suspend fun loadMore(index: Int) {
        if (phase != Phase.Loaded || !hasMore || loadingMore || index < items.size - 6) return
        loadingMore = true
        try {
            val next = fetch(page + 1)
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

    private suspend fun fetch(page: Int): MediaPage = when (source) {
        Source.Trending -> api.trending(page)
        Source.Upcoming -> api.upcoming(page)
        Source.Movies -> api.popularMovies(page)
        Source.Tv -> api.popularTv(page)
    }
}

/** The "For you" chip: the suggestion rows from the server. */
class DiscoverHomeModel(private val api: TipsarrApi) {
    var rows by mutableStateOf<List<SuggestionRow>>(emptyList())
        private set
    var generating by mutableStateOf(false)
        private set
    var phase by mutableStateOf<Phase>(Phase.Idle)
        private set

    suspend fun loadIfNeeded() {
        if (phase == Phase.Idle) refresh()
    }

    suspend fun refresh() {
        if (rows.isEmpty()) phase = Phase.Loading
        try {
            val result = api.suggestions()
            rows = result.rows.filter { it.items.isNotEmpty() }
            generating = result.generating
            phase = Phase.Loaded
        } catch (e: ApiError) {
            if (rows.isEmpty()) phase = Phase.Failed(e)
        }
    }
}

enum class DiscoverChip { ForYou, Trending, Upcoming, Movies, Tv }

/** Everything the Discover tab owns for one account. */
class DiscoverModel(private val api: TipsarrApi) {
    val home = DiscoverHomeModel(api)
    var chip by mutableStateOf(DiscoverChip.ForYou)
    private val lists = HashMap<MediaListModel.Source, MediaListModel>()

    fun list(source: MediaListModel.Source): MediaListModel = lists.getOrPut(source) { MediaListModel(source, api) }

    fun source(chip: DiscoverChip): MediaListModel.Source? = when (chip) {
        DiscoverChip.ForYou -> null
        DiscoverChip.Trending -> MediaListModel.Source.Trending
        DiscoverChip.Upcoming -> MediaListModel.Source.Upcoming
        DiscoverChip.Movies -> MediaListModel.Source.Movies
        DiscoverChip.Tv -> MediaListModel.Source.Tv
    }

    suspend fun refreshCurrent() {
        source(chip)?.let { list(it).refresh() } ?: home.refresh()
    }
}
