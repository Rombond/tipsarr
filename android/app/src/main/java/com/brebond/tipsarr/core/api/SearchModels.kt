package com.brebond.tipsarr.core.api

import kotlinx.serialization.Serializable

@Serializable
data class PersonSummary(val id: Int, val name: String, val department: String? = null, val profilePath: String? = null)

@Serializable
data class SearchPage(val items: List<MediaItem>, val people: List<PersonSummary> = emptyList(), val page: Int = 1, val totalPages: Int = 1) {
    val hasMore: Boolean get() = page < totalPages
}

@Serializable
data class PersonDetail(
    val id: Int,
    val name: String,
    val credits: List<MediaItem> = emptyList(),
    val department: String? = null,
    val biography: String? = null,
    val birthday: String? = null,
    val birthplace: String? = null,
    val deathday: String? = null,
    val profilePath: String? = null,
)

@Serializable
data class RequestCounts(val pending: Int = 0, val approved: Int = 0, val available: Int = 0, val declined: Int = 0, val failed: Int = 0) {
    fun count(filter: RequestFilter): Int = when (filter) {
        RequestFilter.All -> 0
        RequestFilter.Pending -> pending
        RequestFilter.Approved -> approved
        RequestFilter.Available -> available
        RequestFilter.Declined -> declined
        RequestFilter.Failed -> failed
    }
}

enum class RequestFilter(val wire: String) { All("all"), Pending("pending"), Approved("approved"), Available("available"), Declined("declined"), Failed("failed") }

data class RequestPage(val items: List<RequestRecord>, val total: Int)

@Serializable
data class ApproveBody(val qualityProfileId: Int? = null, val rootFolder: String? = null)

@Serializable
data class DeclineBody(val reason: String? = null)
