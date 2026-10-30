package com.brebond.tipsarr.features.teasers

import android.content.Context
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateListOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import com.brebond.tipsarr.core.api.ApiError
import com.brebond.tipsarr.core.api.Availability
import com.brebond.tipsarr.core.api.MediaItem
import com.brebond.tipsarr.core.api.RequestStatus
import com.brebond.tipsarr.core.api.trending
import com.brebond.tipsarr.core.api.Teaser
import com.brebond.tipsarr.core.api.TipsarrApi
import com.brebond.tipsarr.core.api.markTeaserSeen
import com.brebond.tipsarr.core.api.setBlocklisted
import com.brebond.tipsarr.core.api.setWatchlisted
import com.brebond.tipsarr.core.api.teasers
import com.brebond.tipsarr.features.discover.Phase
import kotlinx.coroutines.delay

/**
 * The Teasers feed. Lives in the main view model, so the card the person was on is still there after
 * opening a title and coming back. A title is reported as seen when they swipe past it.
 */
class TeasersModel(private val api: TipsarrApi, private val context: Context) {
    val items = mutableStateListOf<Teaser>()
    var phase by mutableStateOf<Phase>(Phase.Idle)
        private set
    /** True when the server has nothing new left. */
    var done by mutableStateOf(false)
        private set
    var loadingMore by mutableStateOf(false)
        private set
    /** The server answered `teasers_disabled`: the feature was turned off, the tab goes away. */
    var disabled by mutableStateOf(false)
        private set
    /** The card on screen. */
    var page by mutableIntStateOf(0)
    /** A pinch hid the interface: it stays hidden until the next trailer or the eye button (the bottom bar follows). */
    var uiHidden by mutableStateOf(false)
    /** Two fingers are on the picture. */
    var pinching by mutableStateOf(false)

    /**
     * Shown while the first batch is slow (a new person: the server builds the pool and looks trailers up): the trending
     * titles, which is what the server's own default pool is made of. Posters only, the trailers are not ready yet.
     */
    var fallback by mutableStateOf<List<MediaItem>>(emptyList())
        private set

    private val seen = HashSet<String>()

    /** The server is still preparing trailer files: nothing to show yet, ask again shortly. */
    val preparing: Boolean get() = phase == Phase.Loaded && items.isEmpty() && !done

    /** The Teasers screen is composed: it creates and owns its players, [prefetch] does nothing meanwhile. */
    var screenShown = false

    /** Players of the first cards, prepared (buffering, not playing) before the screen is opened. Keyed by [Teaser.id]. */
    private val prepared = HashMap<String, TeaserVideo>()

    suspend fun loadFallback() {
        if (fallback.isNotEmpty()) return
        try {
            fallback = api.trending(1).items.filter { it.availability != Availability.Available && it.requestStatus == null }.take(20)
        } catch (_: ApiError) {}
    }

    suspend fun loadIfNeeded() {
        // A failed background load is tried again when the person opens the tab (unless the feature went away).
        if (phase == Phase.Idle || (phase is Phase.Failed && !disabled)) refresh()
    }

    /**
     * Background warm-up, once the Teasers feature is on: loads the first batch, then prepares the player of the
     * current card and of the next one (paused, so nothing is heard). Safe to call again; call from the main thread.
     */
    suspend fun prefetch() {
        if (screenShown || disabled) return
        loadIfNeeded()
        if (screenShown || phase != Phase.Loaded) return
        val wanted = items.drop(page.coerceIn(0, (items.size - 1).coerceAtLeast(0))).take(2)
        val keep = wanted.map { it.id }.toSet()
        prepared.keys.filter { it !in keep }.forEach { prepared.remove(it)?.release() }
        for (teaser in wanted) {
            if (prepared[teaser.id]?.usable == true) continue
            prepared.remove(teaser.id)?.release()
            prepared[teaser.id] = TeaserVideo(context, api.serverUrl.trimEnd('/') + teaser.videoUrl, api.token) { discard(teaser.id) }
        }
    }

    private fun discard(id: String) { prepared.remove(id)?.release() }

    /** Hands the prepared player of this card over (the caller owns and releases it), or null: create one as usual. */
    fun takePrepared(teaser: Teaser): TeaserVideo? {
        val video = prepared.remove(teaser.id) ?: return null
        if (!video.usable) { video.release(); return null }
        return video
    }

    /** Frees the prepared players (app in the background, feature off, signed out). */
    fun releasePrepared() {
        prepared.values.forEach { it.release() }
        prepared.clear()
    }

    suspend fun refresh() {
        releasePrepared()
        phase = Phase.Loading
        try {
            val feed = api.teasers(BATCH, emptyList())
            items.clear()
            items.addAll(feed.items)
            seen.clear()
            page = 0
            done = feed.done
            phase = Phase.Loaded
        } catch (e: ApiError) {
            if (e.isTeasersDisabled) disabled = true
            phase = Phase.Failed(e)
        }
    }

    /** Loads the next batch when the person is near the end. */
    suspend fun loadMore() {
        if (phase != Phase.Loaded || done || loadingMore) return
        loadingMore = true
        try {
            val feed = api.teasers(BATCH, items.map { it.key })
            val known = items.map { it.id }.toSet()
            items.addAll(feed.items.filter { it.id !in known })
            if (feed.done) done = true // an empty answer with done == false: trailers are being prepared, retry on the next swipe
        } catch (e: ApiError) {
            if (e.isTeasersDisabled) disabled = true // otherwise: try again on the next swipe
        } finally {
            loadingMore = false
        }
    }

    /** While [preparing], asks the server again every [POLL_MS] ms until cards arrive or the feed is done. Cancel to stop. */
    suspend fun pollWhilePreparing() {
        while (preparing) {
            delay(POLL_MS)
            try {
                val feed = api.teasers(BATCH, emptyList())
                items.addAll(feed.items)
                if (feed.done) done = true
            } catch (e: ApiError) {
                if (e.isTeasersDisabled) { disabled = true; return }
            }
        }
    }

    /** Reports the card as swiped past (once). Best effort. */
    suspend fun markSeen(teaser: Teaser) {
        if (!seen.add(teaser.id)) return
        try { api.markTeaserSeen(teaser.type, teaser.tmdbId) } catch (_: ApiError) { seen.remove(teaser.id) }
    }

    /** "Not interested": blocklist the title and drop the card (the next one slides in). */
    suspend fun hide(teaser: Teaser) {
        api.setBlocklisted(true, teaser.type, teaser.tmdbId)
        items.remove(teaser)
        if (page >= items.size) page = (items.size - 1).coerceAtLeast(0)
    }

    /** The trailer file cannot be played: skip it for good without any message. */
    suspend fun skip(teaser: Teaser) {
        markSeen(teaser)
        items.remove(teaser)
        if (page >= items.size) page = (items.size - 1).coerceAtLeast(0)
    }

    suspend fun toggleWatchlist(teaser: Teaser) {
        val on = !teaser.watchlisted
        api.setWatchlisted(on, teaser.type, teaser.tmdbId)
        replace(teaser) { it.copy(watchlisted = on) }
    }

    /** A request was made from the card: show it as requested, or approved when it was approved at once. */
    fun requested(teaser: Teaser, approved: Boolean) =
        replace(teaser) { it.copy(requestStatus = if (approved) RequestStatus.Approved else RequestStatus.Pending) }

    private fun replace(teaser: Teaser, change: (Teaser) -> Teaser) {
        val at = items.indexOfFirst { it.id == teaser.id }
        if (at >= 0) items[at] = change(items[at])
    }

    private val ApiError.isTeasersDisabled: Boolean get() = this is ApiError.Http && status == 404 && code == "teasers_disabled"

    private companion object {
        const val BATCH = 8
        const val POLL_MS = 8_000L
    }
}
