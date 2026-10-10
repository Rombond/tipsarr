package com.brebond.tipsarr.features.profile

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import com.brebond.tipsarr.core.api.LibraryFilters
import com.brebond.tipsarr.core.api.RequestRecord
import com.brebond.tipsarr.core.api.StatsTop
import com.brebond.tipsarr.core.api.TipsarrApi
import com.brebond.tipsarr.core.api.WatchedFilter
import com.brebond.tipsarr.core.api.library
import com.brebond.tipsarr.core.api.requests
import com.brebond.tipsarr.core.api.RequestFilter
import com.brebond.tipsarr.core.api.statsAll
import com.brebond.tipsarr.core.api.watchlist
import kotlinx.coroutines.async
import kotlinx.coroutines.coroutineScope

/** Numbers and recent requests shown on the Profile tab. */
class ProfileModel(private val api: TipsarrApi) {
    var requestCount by mutableStateOf<Int?>(null)
        private set
    var watchlistCount by mutableStateOf<Int?>(null)
        private set
    var watchedCount by mutableStateOf<Int?>(null)
        private set
    var recent by mutableStateOf<List<RequestRecord>>(emptyList())
        private set

    /** Everything this person watched, most time first (movies and shows mixed). */
    var topWatched by mutableStateOf<List<StatsTop>>(emptyList())
        private set

    suspend fun load() = coroutineScope {
        val mine = async { runCatching { api.requests(RequestFilter.All, 0, 4) }.getOrNull() }
        val watchlist = async { runCatching { api.watchlist() }.getOrNull() }
        val watched = async { runCatching { api.library(LibraryFilters(watched = WatchedFilter.Yes), 1, 1) }.getOrNull() }
        val stats = async { runCatching { api.statsAll() }.getOrNull() }
        mine.await()?.let { requestCount = it.total; recent = it.items }
        watchlist.await()?.let { watchlistCount = it.size }
        watched.await()?.let { watchedCount = it.total }
        stats.await()?.let { topWatched = it.top }
    }
}
