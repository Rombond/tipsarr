package com.brebond.tipsarr.core.api

import kotlinx.serialization.Serializable
import kotlinx.serialization.builtins.ListSerializer
import kotlinx.serialization.builtins.MapSerializer
import kotlinx.serialization.builtins.serializer

enum class RatingSource(val wire: String) {
    Tmdb("tmdb"), Imdb("imdb"), Metacritic("metacritic"), RottenTomatoes("rottenTomatoes");

    companion object {
        fun from(wire: String?): RatingSource = entries.firstOrNull { it.wire == wire } ?: Tmdb
    }
}

@Serializable
data class PrefsBody(val language: String? = null, val ratingSource: String? = null, val region: String? = null)

@Serializable
data class DeviceSession(
    val id: String,
    val platform: String = "web",
    val deviceName: String = "",
    val appVersion: String = "",
    val lastSeenAt: Long = 0,
    val current: Boolean = false,
)

@Serializable
data class StatsTop(
    val type: MediaType,
    val tmdbId: Int,
    val title: String,
    val plays: Int = 0,
    val hours: Double = 0.0,
    val year: Int? = null,
    val posterUrl: String? = null,
) {
    val posterPath: String? get() = posterUrl?.takeIf { it.isNotEmpty() }
}

@Serializable
data class StatsReport(val top: List<StatsTop> = emptyList())

/** Saves one or more preferences; returns the updated profile. Pass an empty string to clear a value. */
suspend fun TipsarrApi.updatePreferences(language: String? = null, ratingSource: RatingSource? = null, region: String? = null): Profile =
    patch("/me", PrefsBody(language, ratingSource?.wire, region), PrefsBody.serializer(), Profile.serializer())

suspend fun TipsarrApi.sessions(): List<DeviceSession> = get("/me/sessions", ListSerializer(DeviceSession.serializer()))

suspend fun TipsarrApi.revokeSession(id: String) = delete("/me/sessions/$id")

suspend fun TipsarrApi.revokeOtherSessions() = delete("/me/sessions")

suspend fun TipsarrApi.uploadAvatar(jpeg: ByteArray) = postJpeg("/me/avatar", jpeg)

suspend fun TipsarrApi.deleteAvatar() = delete("/me/avatar")

/** What this person watched, most time first (`period=all`). */
suspend fun TipsarrApi.statsAll(): StatsReport = get("/stats?period=all", StatsReport.serializer())

/** Scores for several movies at once, keyed by TMDB id. */
suspend fun TipsarrApi.movieScores(ids: List<Int>, source: RatingSource): Map<Int, MovieScores> {
    val map = get("/ratings/movies?ids=${ids.joinToString(",")}&source=${source.wire}", MapSerializer(String.serializer(), MovieScores.serializer()))
    return map.mapNotNull { (key, value) -> key.toIntOrNull()?.let { it to value } }.toMap()
}
