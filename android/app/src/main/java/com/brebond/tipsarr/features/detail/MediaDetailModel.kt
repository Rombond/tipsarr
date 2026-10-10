package com.brebond.tipsarr.features.detail

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import com.brebond.tipsarr.core.api.ApiError
import com.brebond.tipsarr.core.api.IssueKind
import com.brebond.tipsarr.core.api.MediaDetail
import com.brebond.tipsarr.core.api.MediaType
import com.brebond.tipsarr.core.api.RatingsSummary
import com.brebond.tipsarr.core.api.RequestOptions
import com.brebond.tipsarr.core.api.RequestRecord
import com.brebond.tipsarr.core.api.SeasonInfo
import com.brebond.tipsarr.core.api.TipsarrApi
import com.brebond.tipsarr.core.api.TitleFlags
import com.brebond.tipsarr.core.api.Availability
import com.brebond.tipsarr.core.api.createRequest
import com.brebond.tipsarr.core.api.deleteRequest
import com.brebond.tipsarr.core.api.detail
import com.brebond.tipsarr.core.api.flags
import com.brebond.tipsarr.core.api.movieScores
import com.brebond.tipsarr.core.api.myRequests
import com.brebond.tipsarr.core.api.reportIssue
import com.brebond.tipsarr.core.api.requestOptions
import com.brebond.tipsarr.core.api.retryRequest
import com.brebond.tipsarr.core.api.setBlocklisted
import com.brebond.tipsarr.core.api.setWatchlisted
import com.brebond.tipsarr.ui.RequestState
import kotlinx.coroutines.async
import kotlinx.coroutines.coroutineScope

/** Navigation value for the detail screen. */
data class MediaRoute(val type: MediaType, val tmdbId: Int, val title: String)

/** What the main button of the detail screen does. */
sealed interface DetailAction {
    data object Request : DetailAction
    data class InProgress(val state: RequestState, val cancellable: Boolean) : DetailAction
    data class Available(val watchUrl: String?) : DetailAction
    data object RequestAgain : DetailAction
    data class Failed(val canRetry: Boolean) : DetailAction
}

class MediaDetailModel(
    val route: MediaRoute,
    val api: TipsarrApi,
    val isAdmin: Boolean,
    /** `userFolderChoice` from `/status`: non-admins may pick the folder when true. */
    userFolderChoice: Boolean,
) {
    sealed interface Phase {
        data object Loading : Phase
        data object Loaded : Phase
        data class Failed(val error: ApiError) : Phase
    }

    var phase by mutableStateOf<Phase>(Phase.Loading)
        private set
    var detail by mutableStateOf<MediaDetail?>(null)
        private set
    var ratings by mutableStateOf(RatingsSummary())
        private set
    var flags by mutableStateOf(TitleFlags())
        private set

    /** The signed-in user's newest request for this title. */
    var request by mutableStateOf<RequestRecord?>(null)
        private set

    /** Seasons already covered by one of the user's requests (open or available). */
    var coveredSeasons by mutableStateOf<Set<Int>>(emptySet())
        private set
    var options by mutableStateOf<RequestOptions?>(null)
        private set
    var busy by mutableStateOf(false)
        private set

    val canChooseFolder: Boolean = isAdmin || userFolderChoice

    suspend fun load() {
        if (detail == null) phase = Phase.Loading
        val loaded = try {
            api.detail(route.type, route.tmdbId)
        } catch (e: ApiError) {
            if (detail == null) phase = Phase.Failed(e)
            return
        }
        detail = loaded
        ratings = ratings.copy(tmdb = loaded.voteAverage.takeIf { it > 0 })
        phase = Phase.Loaded
        // The rest is optional: the screen already works without it.
        coroutineScope {
            val scores = async { if (route.type == MediaType.Movie) runCatching { api.movieScores(route.tmdbId) }.getOrNull() else null }
            val titleFlags = async { runCatching { api.flags(route.type, route.tmdbId) }.getOrNull() }
            val mine = async { runCatching { api.myRequests() }.getOrNull() }
            scores.await()?.let { ratings = ratings.copy(imdb = it.imdb, rottenTomatoes = it.rottenTomatoes, metacritic = it.metacritic) }
            titleFlags.await()?.let { flags = it }
            mine.await()?.let { apply(it) }
        }
    }

    private fun apply(all: List<RequestRecord>) {
        val forTitle = all.filter { it.type == route.type && it.tmdbId == route.tmdbId }
        request = forTitle.maxByOrNull { it.createdAt }
        coveredSeasons = forTitle.filter { it.state != RequestState.Declined && it.state != RequestState.Failed }.flatMap { it.seasons }.toSet()
    }

    // State of the main button

    val action: DetailAction
        get() {
            val detail = detail ?: return DetailAction.Request
            if (detail.availability == Availability.Available) return DetailAction.Available(detail.watchUrl)
            request?.let { request ->
                if (request.isOpen) return DetailAction.InProgress(request.state, cancellable = true)
                if (request.state == RequestState.Declined) return DetailAction.RequestAgain
                if (request.state == RequestState.Failed) return DetailAction.Failed(canRetry = isAdmin)
            }
            if (detail.requestOpen) return DetailAction.InProgress(RequestState.Requested, cancellable = false)
            return DetailAction.Request
        }

    /** The badge next to the title. */
    val badge: RequestState?
        get() = when (val action = action) {
            is DetailAction.Available -> RequestState.Available
            is DetailAction.InProgress -> action.state
            DetailAction.RequestAgain -> RequestState.Declined
            is DetailAction.Failed -> RequestState.Failed
            DetailAction.Request -> if (detail?.availability == Availability.Partial) RequestState.Partial else null
        }

    val regularSeasons: List<SeasonInfo> get() = detail?.seasons.orEmpty().filter { it.number > 0 }

    // Actions

    /** Loads quality profiles. Returns whether the request needs a sheet (a choice to make). */
    suspend fun prepareRequest(): Boolean {
        busy = true
        try {
            options = api.requestOptions(route.type)
        } finally {
            busy = false
        }
        return options != null || (route.type == MediaType.Tv && regularSeasons.size > 1)
    }

    suspend fun submitRequest(seasons: List<Int>?, profileId: Int?, folder: String?): RequestRecord {
        busy = true
        try {
            val record = api.createRequest(route.type, route.tmdbId, seasons, profileId, folder)
            request = record
            coveredSeasons = coveredSeasons + record.seasons
            detail = detail?.copy(requestStatus = com.brebond.tipsarr.core.api.RequestStatus.Pending)
            return record
        } finally {
            busy = false
        }
    }

    suspend fun cancelRequest() {
        val current = request ?: return
        busy = true
        try {
            api.deleteRequest(current.id)
            request = null
            coveredSeasons = coveredSeasons - current.seasons.toSet()
            detail = detail?.copy(requestStatus = null)
        } finally {
            busy = false
        }
    }

    suspend fun retryRequest() {
        val current = request ?: return
        busy = true
        try {
            request = api.retryRequest(current.id)
        } finally {
            busy = false
        }
    }

    suspend fun toggleWatchlist() {
        val on = !flags.watchlisted
        api.setWatchlisted(on, route.type, route.tmdbId)
        flags = flags.copy(watchlisted = on)
    }

    suspend fun toggleHidden() {
        val on = !flags.blocklisted
        api.setBlocklisted(on, route.type, route.tmdbId)
        flags = flags.copy(blocklisted = on)
    }

    suspend fun report(kind: IssueKind, message: String) = api.reportIssue(route.type, route.tmdbId, kind, message)
}
