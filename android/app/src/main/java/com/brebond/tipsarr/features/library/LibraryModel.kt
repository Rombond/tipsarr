package com.brebond.tipsarr.features.library

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import com.brebond.tipsarr.core.api.ApiError
import com.brebond.tipsarr.core.api.LibraryFacets
import com.brebond.tipsarr.core.api.LibraryFilters
import com.brebond.tipsarr.core.api.LibraryItem
import com.brebond.tipsarr.core.api.TipsarrApi
import com.brebond.tipsarr.core.api.library
import com.brebond.tipsarr.core.api.libraryFacets
import com.brebond.tipsarr.features.discover.Phase

class LibraryModel(private val api: TipsarrApi) {
    var filters by mutableStateOf(LibraryFilters())
    var phase by mutableStateOf<Phase>(Phase.Idle)
        private set
    var items by mutableStateOf<List<LibraryItem>>(emptyList())
        private set
    var total by mutableStateOf(0)
        private set
    var facets by mutableStateOf<LibraryFacets?>(null)
        private set
    var loadingMore by mutableStateOf(false)
        private set
    private var page = 0
    private var hasMore = false

    /** True when nothing is filtered and the library has no title: it was never synced. */
    val neverSynced: Boolean get() = phase == Phase.Loaded && items.isEmpty() && filters == LibraryFilters()

    suspend fun loadFacets() {
        if (facets == null) facets = runCatching { api.libraryFacets() }.getOrNull()
    }

    suspend fun reload() {
        if (items.isEmpty()) phase = Phase.Loading
        val current = filters
        try {
            val result = api.library(current, 1)
            if (current != filters) return
            items = result.items
            total = result.total
            page = result.page
            hasMore = result.hasMore
            phase = Phase.Loaded
        } catch (e: ApiError) {
            if (current == filters && items.isEmpty()) phase = Phase.Failed(e)
        }
    }

    suspend fun loadMore(index: Int) {
        if (phase != Phase.Loaded || !hasMore || loadingMore || index < items.size - 9) return
        loadingMore = true
        val current = filters
        try {
            val next = api.library(current, page + 1)
            if (current == filters) {
                val known = items.mapTo(HashSet()) { it.id }
                items = items + next.items.filter { it.id !in known }
                page = next.page
                hasMore = next.hasMore
            }
        } catch (_: ApiError) {
        } finally {
            loadingMore = false
        }
    }
}
