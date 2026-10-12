package com.brebond.tipsarr.core.api

import kotlinx.serialization.Serializable

/** One card of the Teasers feed: a title with the trailer file the server plays ([videoUrl], relative to the server). */
@Serializable
data class Teaser(
    val type: MediaType,
    val tmdbId: Int,
    val title: String,
    val trailerKey: String, // identifier only
    val voteAverage: Double = 0.0,
    val availability: Availability = Availability.None,
    val posterPath: String? = null,
    val backdropPath: String? = null,
    val overview: String? = null,
    val releaseDate: String? = null,
    val requestStatus: RequestStatus? = null,
    val genreIds: List<Int> = emptyList(),
    val watchlisted: Boolean = false,
    val videoUrl: String,
    val videoHeight: Int = 0,
) {
    val id: String get() = "${type.name}-$tmdbId"

    /** The key the server uses to exclude titles the app already holds (`movie:603`). */
    val key: String get() = "${if (type == MediaType.Movie) "movie" else "tv"}:$tmdbId"

    val item: MediaItem
        get() = MediaItem(type, tmdbId, title, voteAverage, availability, posterPath, backdropPath, overview, releaseDate, requestStatus, genreIds)
}

@Serializable
data class TeaserFeed(val items: List<Teaser> = emptyList(), val done: Boolean = false)

@Serializable
private data class TeaserSeenBody(val type: MediaType, val tmdbId: Int)

suspend fun TipsarrApi.teasers(limit: Int, exclude: List<String>): TeaserFeed {
    val skip = if (exclude.isEmpty()) "" else "&exclude=" + java.net.URLEncoder.encode(exclude.joinToString(","), "UTF-8")
    return get("/teasers?limit=$limit$skip", TeaserFeed.serializer())
}

suspend fun TipsarrApi.markTeaserSeen(type: MediaType, tmdbId: Int) =
    postUnit("/teasers/seen", TeaserSeenBody(type, tmdbId), TeaserSeenBody.serializer())
