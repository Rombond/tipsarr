package com.brebond.tipsarr.core.api

import com.brebond.tipsarr.ui.RequestState
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
enum class MediaType {
    @SerialName("movie") Movie,
    @SerialName("tv") Tv,
}

@Serializable
enum class Availability {
    @SerialName("none") None,
    @SerialName("partial") Partial,
    @SerialName("available") Available,
}

@Serializable
enum class RequestStatus {
    @SerialName("pending") Pending,
    @SerialName("approved") Approved,
}

/** A title in a list (discover, search, suggestions). */
@Serializable
data class MediaItem(
    val type: MediaType,
    val tmdbId: Int,
    val title: String,
    val voteAverage: Double = 0.0,
    val availability: Availability = Availability.None,
    val posterPath: String? = null,
    val backdropPath: String? = null,
    val overview: String? = null,
    val releaseDate: String? = null,
    val requestStatus: RequestStatus? = null,
) {
    val id: String get() = "${type.name}-$tmdbId"
    val releaseYear: String? get() = releaseDate?.take(4)?.takeIf { it.isNotEmpty() }

    /** The badge on the poster: what the library has, else the open request. */
    val state: RequestState?
        get() = when {
            availability == Availability.Available -> RequestState.Available
            availability == Availability.Partial -> RequestState.Partial
            requestStatus == RequestStatus.Approved -> RequestState.Approved
            requestStatus == RequestStatus.Pending -> RequestState.Requested
            else -> null
        }
}

@Serializable
data class MediaPage(val items: List<MediaItem>, val page: Int, val totalPages: Int) {
    val hasMore: Boolean get() = page < totalPages
}

@Serializable
data class SuggestionRow(val id: String, val title: String, val items: List<MediaItem> = emptyList())

@Serializable
data class SuggestionResult(val rows: List<SuggestionRow>, val generating: Boolean = false)
