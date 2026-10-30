package com.brebond.tipsarr.features.requests

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import com.brebond.tipsarr.core.api.ApiError
import com.brebond.tipsarr.core.api.RequestCounts
import com.brebond.tipsarr.core.api.RequestFilter
import com.brebond.tipsarr.core.api.RequestRecord
import com.brebond.tipsarr.core.api.TipsarrApi
import com.brebond.tipsarr.core.api.approve
import com.brebond.tipsarr.core.api.decline
import com.brebond.tipsarr.core.api.deleteRequest
import com.brebond.tipsarr.core.api.request
import com.brebond.tipsarr.core.api.Permission
import com.brebond.tipsarr.core.api.Profile
import com.brebond.tipsarr.core.api.requestCounts
import com.brebond.tipsarr.core.api.updateRequestOptions
import com.brebond.tipsarr.core.api.RequestOptions
import com.brebond.tipsarr.core.api.MediaType
import com.brebond.tipsarr.core.api.requestOptions
import com.brebond.tipsarr.core.api.requests
import com.brebond.tipsarr.core.api.retryRequest
import com.brebond.tipsarr.features.discover.Phase
import com.brebond.tipsarr.ui.RequestState
import kotlinx.coroutines.async
import kotlinx.coroutines.coroutineScope

class RequestsModel(private val api: TipsarrApi, private val me: () -> Profile) {
    /** May approve, decline, retry and delete any request. */
    val canManage: Boolean get() = me().can(Permission.MANAGE_REQUESTS)

    /** Sees everyone's requests, so the requester is shown. */
    val canViewAll: Boolean get() = me().can(Permission.REQUEST_VIEW)

    var filter by mutableStateOf(RequestFilter.All)
    var phase by mutableStateOf<Phase>(Phase.Idle)
        private set
    var items by mutableStateOf<List<RequestRecord>>(emptyList())
        private set
    var counts by mutableStateOf(RequestCounts())
        private set
    var busy by mutableStateOf<Set<String>>(emptySet())
        private set
    var loadingMore by mutableStateOf(false)
        private set
    private var total = 0

    suspend fun load(forFilter: RequestFilter = filter) {
        if (phase == Phase.Idle || phase == Phase.Loading) phase = Phase.Loading
        coroutineScope {
            val fresh = async { runCatching { api.requestCounts() }.getOrNull() }
            try {
                val page = api.requests(forFilter, 0)
                if (filter != forFilter) return@coroutineScope
                items = page.items
                total = page.total
                phase = Phase.Loaded
            } catch (e: ApiError) {
                if (filter == forFilter && items.isEmpty()) phase = Phase.Failed(e)
            }
            fresh.await()?.let { counts = it }
        }
    }

    suspend fun loadMore(index: Int) {
        if (phase != Phase.Loaded || loadingMore || items.size >= total || index < items.size - 4) return
        loadingMore = true
        val current = filter
        try {
            val page = api.requests(current, items.size)
            if (filter == current) {
                val known = items.mapTo(HashSet()) { it.id }
                items = items + page.items.filter { it.id !in known }
                total = page.total
            }
        } catch (_: ApiError) {
        } finally {
            loadingMore = false
        }
    }

    // Actions: return the updated record, the list follows.

    suspend fun approve(record: RequestRecord) = act(record) { api.approve(record.id) }
    suspend fun decline(record: RequestRecord, reason: String) = act(record) { api.decline(record.id, reason) }
    suspend fun retry(record: RequestRecord) = act(record) { api.retryRequest(record.id) }

    suspend fun delete(record: RequestRecord) {
        busy = busy + record.id
        try {
            api.deleteRequest(record.id)
            items = items.filter { it.id != record.id }
            total = maxOf(0, total - 1)
            refreshCounts()
        } finally {
            busy = busy - record.id
        }
    }

    /** Same rule as the web: the requester or an admin may change the quality profile / folder until the request is sent. */
    fun canEdit(record: RequestRecord): Boolean =
        !record.imported && (record.state == RequestState.Requested || record.state == RequestState.Failed) &&
            (canManage || (record.requestedById != null && record.requestedById == me().id))

    suspend fun options(type: MediaType): RequestOptions? = api.requestOptions(type)

    /** Only what changed from the instance defaults is sent (null: leave as is). */
    suspend fun updateOptions(record: RequestRecord, profileId: Int?, folder: String?) =
        act(record) { api.updateRequestOptions(record.id, profileId, folder) }

    suspend fun fresh(id: String): RequestRecord? = runCatching { api.request(id) }.getOrNull()

    private suspend fun act(record: RequestRecord, work: suspend () -> RequestRecord): RequestRecord {
        busy = busy + record.id
        try {
            val updated = work()
            val index = items.indexOfFirst { it.id == record.id }
            if (index >= 0) {
                items = if (matches(updated, filter)) items.toMutableList().also { it[index] = updated } else items.filterIndexed { i, _ -> i != index }
            }
            refreshCounts()
            return updated
        } finally {
            busy = busy - record.id
        }
    }

    suspend fun refreshCounts() {
        runCatching { api.requestCounts() }.getOrNull()?.let { counts = it }
    }

    private fun matches(record: RequestRecord, filter: RequestFilter): Boolean = when (filter) {
        RequestFilter.All -> true
        RequestFilter.Pending -> record.state == RequestState.Requested
        RequestFilter.Approved -> record.state in listOf(RequestState.Approved, RequestState.Searching, RequestState.Downloading)
        RequestFilter.Available -> record.state == RequestState.Available
        RequestFilter.Declined -> record.state == RequestState.Declined
        RequestFilter.Failed -> record.state == RequestState.Failed
    }

    /** People who manage requests can delete anything; owners only unfinished requests (the server enforces it too). */
    fun canDelete(record: RequestRecord): Boolean = canManage || record.isOpen
}
