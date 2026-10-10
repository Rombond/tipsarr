package com.brebond.tipsarr.core.api

import kotlinx.serialization.Serializable

// Issues

@Serializable
data class IssueUser(val id: String = "", val name: String = "")

@Serializable
data class IssueView(
    val id: String,
    val type: MediaType,
    val tmdbId: Int,
    val title: String,
    val kind: IssueKind,
    val status: String,
    val createdBy: IssueUser = IssueUser(),
    val createdAt: Long = 0,
    val commentCount: Int = 0,
    val posterPath: String? = null,
    val season: Int? = null,
    val episode: Int? = null,
    val resolvedBy: IssueUser? = null,
) {
    val isOpen: Boolean get() = status == "open"
}

@Serializable
data class IssueComment(val id: String, val message: String, val user: IssueUser = IssueUser(), val createdAt: Long = 0)

@Serializable
data class IssueThread(
    val id: String,
    val type: MediaType,
    val tmdbId: Int,
    val title: String,
    val kind: IssueKind,
    val status: String,
    val createdBy: IssueUser = IssueUser(),
    val createdAt: Long = 0,
    val commentCount: Int = 0,
    val comments: List<IssueComment> = emptyList(),
    val posterPath: String? = null,
    val season: Int? = null,
    val episode: Int? = null,
    val resolvedBy: IssueUser? = null,
) {
    val isOpen: Boolean get() = status == "open"
}

@Serializable
data class IssueList(val items: List<IssueView>, val total: Int = 0)

@Serializable
data class IssueCounts(val open: Int = 0, val resolved: Int = 0)

@Serializable
data class CommentBody(val message: String)

enum class IssueFilter(val wire: String) { Open("open"), Resolved("resolved"), All("all") }

// Users

@Serializable
data class UserStats(
    val requests: Int = 0, val movies: Int = 0, val shows: Int = 0, val pending: Int = 0, val approved: Int = 0,
    val available: Int = 0, val declined: Int = 0, val failed: Int = 0, val watchlist: Int = 0, val watched: Int = 0,
)

@Serializable
data class UserDetail(
    val id: String,
    val name: String,
    val role: String,
    val region: String = "",
    val language: String = "",
    val ratingSource: String = "",
    val createdAt: Long = 0,
    val lastLoginAt: Long = 0,
    val stats: UserStats = UserStats(),
) {
    val isAdmin: Boolean get() = role == "admin"
}

@Serializable
data class RoleBody(val role: String)

// Sync

@Serializable
data class JobStatus(
    val name: String,
    val running: Boolean = false,
    val status: String = "",
    val message: String = "",
    val lastStartedAt: Long = 0,
    val lastFinishedAt: Long = 0,
    val everySeconds: Int = 0,
)

@Serializable
data class SyncStatus(val canSync: Boolean = false, val jobs: List<JobStatus> = emptyList(), val movies: Int = 0, val shows: Int = 0)

// Stats (the profile only needs `top`, the Stats screen everything)

@Serializable
data class StatsBucket(val name: String, val hours: Double = 0.0, val titles: Int = 0)

@Serializable
data class StatsMonth(val month: String, val hours: Double = 0.0, val plays: Int = 0)

@Serializable
data class StatsTotals(val titles: Int = 0, val movies: Int = 0, val shows: Int = 0, val plays: Int = 0, val hours: Double = 0.0)

@Serializable
data class StatsRequests(val made: Int = 0, val available: Int = 0, val declined: Int = 0)

@Serializable
data class StatsPlugin(val hint: Boolean = false)

@Serializable
data class FullStats(
    val totals: StatsTotals = StatsTotals(),
    val requests: StatsRequests = StatsRequests(),
    val genres: List<StatsBucket> = emptyList(),
    val decades: List<StatsBucket> = emptyList(),
    val months: List<StatsMonth> = emptyList(),
    val weekdays: List<Double> = emptyList(),
    val hoursOfDay: List<Double> = emptyList(),
    val top: List<StatsTop> = emptyList(),
    val topMovies: List<StatsTop> = emptyList(),
    val topShows: List<StatsTop> = emptyList(),
    val source: String = "estimate",
    val plugin: StatsPlugin = StatsPlugin(),
) {
    val exact: Boolean get() = source == "plugin"
}

enum class StatsPeriod(val wire: String) { Days30("30d"), Months12("12m"), All("all") }
