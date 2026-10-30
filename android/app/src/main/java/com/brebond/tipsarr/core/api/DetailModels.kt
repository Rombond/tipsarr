package com.brebond.tipsarr.core.api

import com.brebond.tipsarr.ui.RequestState
import kotlinx.serialization.Serializable

@Serializable
data class Genre(val id: Int, val name: String)

@Serializable
data class SeasonInfo(val number: Int, val name: String, val episodeCount: Int = 0, val airDate: String? = null, val libraryEpisodes: Int = 0, val monitored: Boolean = false) {
    val id: Int get() = number
    /** The library holds every announced episode: nothing left to request. */
    val isComplete: Boolean get() = episodeCount > 0 && libraryEpisodes >= episodeCount
    /** Badges of the season: in the library, partly in it, and/or tracked by Sonarr (shown like an approved request). */
    val badges: List<RequestState> get() = when {
        isComplete -> listOf(RequestState.Available)
        else -> listOfNotNull(RequestState.Partial.takeIf { libraryEpisodes > 0 }, RequestState.Approved.takeIf { monitored })
    }
}

@Serializable
data class CastMember(val id: Int, val name: String, val character: String? = null, val profilePath: String? = null)

/** Full detail of a title (the fields the screen shows; the API sends more). */
@Serializable
data class MediaDetail(
    val type: MediaType,
    val tmdbId: Int,
    val title: String,
    val availability: Availability = Availability.None,
    val tagline: String? = null,
    val overview: String? = null,
    val posterPath: String? = null,
    val backdropPath: String? = null,
    val releaseDate: String? = null,
    val runtimeMinutes: Int? = null,
    val voteAverage: Double = 0.0,
    val genres: List<Genre> = emptyList(),
    val status: String? = null,
    val originalLanguage: String? = null,
    val requestStatus: RequestStatus? = null,
    /** Radarr monitors the movie and looks for it. */
    val monitored: Boolean = false,
    val seasons: List<SeasonInfo>? = null,
    val numberOfSeasons: Int? = null,
    val studios: List<String>? = null,
    val cast: List<CastMember> = emptyList(),
    val recommendations: List<MediaItem> = emptyList(),
    val watchUrl: String? = null,
    val trailerKey: String? = null,
    val imdbId: String? = null,
) {
    val year: String? get() = releaseDate?.takeIf { it.length >= 4 }?.take(4)
    val requestOpen: Boolean get() = requestStatus != null
}

@Serializable
data class Score(val value: Double, val votes: Int? = null)

@Serializable
data class MovieScores(val imdb: Score? = null, val metacritic: Score? = null, val rottenTomatoes: Score? = null, val rottenTomatoesUrl: String? = null)

/** Scores for the ratings strip. TMDB comes with the detail, the others only exist for movies. */
data class RatingsSummary(val tmdb: Double? = null, val imdb: Double? = null, val rottenTomatoes: Double? = null, val metacritic: Double? = null, val rottenTomatoesUrl: String? = null)

@Serializable
data class TitleFlags(val watchlisted: Boolean = false, val blocklisted: Boolean = false)

@Serializable
data class QualityProfile(val id: Int, val name: String)

@Serializable
data class RootFolder(val id: Int, val path: String, val freeSpace: Long = 0)

@Serializable
data class RequestOptions(
    val instanceName: String = "",
    val profiles: List<QualityProfile> = emptyList(),
    val qualityProfileId: Int = 0,
    val rootFolders: List<RootFolder> = emptyList(),
    val rootFolder: String = "",
)

@Serializable
data class CreateRequestBody(val type: MediaType, val tmdbId: Int, val seasons: List<Int>? = null, val qualityProfileId: Int? = null, val rootFolder: String? = null)

@Serializable
data class UserRef(val id: String = "", val name: String = "")

@Serializable
data class Progress(val percent: Int, val etaSeconds: Int = 0)

@Serializable
data class SeasonProgress(val season: Int, val percent: Int)

/** A request as the server sends it (`View`). */
@Serializable
data class RequestView(
    val id: String,
    val type: MediaType,
    val tmdbId: Int = 0,
    val title: String = "",
    val stage: String? = null,
    val status: String,
    val posterPath: String? = null,
    val seasons: List<Int>? = null,
    val progress: Progress? = null,
    val seasonProgress: List<SeasonProgress>? = null,
    val declineReason: String? = null,
    val error: String? = null,
    val requestedBy: UserRef = UserRef(),
    val decidedBy: UserRef? = null,
    val dryRun: Boolean = false,
    val createdAt: Long = 0,
    val qualityProfileId: Int? = null,
    val rootFolder: String? = null,
    /** Set when the request was imported from what Radarr/Sonarr already monitor. */
    val source: String? = null,
)

data class RequestRecord(
    val id: String,
    val type: MediaType,
    val tmdbId: Int,
    val title: String,
    val posterPath: String?,
    val state: RequestState,
    val seasons: List<Int>,
    val progressPercent: Int?,
    val etaSeconds: Int?,
    val seasonProgress: Map<Int, Int>,
    val declineReason: String?,
    val error: String?,
    val requestedBy: String?,
    val decidedBy: String?,
    val dryRun: Boolean,
    val createdAt: Long,
    val requestedById: String? = null,
    val qualityProfileId: Int? = null,
    val rootFolder: String? = null,
    val imported: Boolean = false,
) {
    /** The owner may still cancel until the title is available. */
    val isOpen: Boolean get() = state != RequestState.Available && state != RequestState.Declined && state != RequestState.Failed
}

fun RequestView.toRecord() = RequestRecord(
    id = id, type = type, tmdbId = tmdbId, title = title, posterPath = posterPath,
    state = RequestState.from(stage, status),
    seasons = seasons.orEmpty(),
    progressPercent = progress?.percent, etaSeconds = progress?.etaSeconds,
    seasonProgress = seasonProgress.orEmpty().associate { it.season to it.percent },
    declineReason = declineReason?.takeIf { it.isNotEmpty() },
    error = error?.takeIf { it.isNotEmpty() },
    requestedBy = requestedBy.name, decidedBy = decidedBy?.name?.takeIf { it.isNotEmpty() },
    dryRun = dryRun, createdAt = createdAt,
    requestedById = requestedBy.id.takeIf { it.isNotEmpty() },
    qualityProfileId = qualityProfileId?.takeIf { it != 0 }, rootFolder = rootFolder?.takeIf { it.isNotEmpty() },
    imported = !source.isNullOrEmpty(),
)

@Serializable
data class RequestList(val items: List<RequestView>, val total: Int = 0)

@Serializable
data class Episode(
    val number: Int,
    val name: String,
    val voteAverage: Double = 0.0,
    val overview: String? = null,
    val airDate: String? = null,
    val runtimeMinutes: Int? = null,
    val stillPath: String? = null,
    /** "available", "monitored" or "missing" as Sonarr knows it; null when no Sonarr could tell. */
    val state: String? = null,
) {
    val id: Int get() = number
    /** A file exists, or Sonarr is looking for it; missing and unknown show no badge. */
    val badge: RequestState? get() = when (state) { "available" -> RequestState.Available; "monitored" -> RequestState.Approved; else -> null }
}

@Serializable
data class SeasonDetail(val number: Int, val name: String, val episodes: List<Episode>)

@Serializable
data class WatchFlagBody(val tmdbId: Int, val type: MediaType)

@Serializable
enum class IssueKind { video, audio, subtitles, other }

@Serializable
data class CreateIssueBody(val kind: IssueKind, val message: String, val tmdbId: Int, val type: MediaType)
