package com.brebond.tipsarr.core.live

import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.compositionLocalOf
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateMapOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

/** Progress of a download pushed by the server. */
data class LiveProgress(val percent: Int, val etaSeconds: Int)

/**
 * What changed on the server, as counters screens can watch with [OnTick]. The stream runs only while the app is
 * in the foreground and a person is signed in.
 */
class LiveUpdates(private val scope: CoroutineScope) {
    /** A request changed (created, approved, declined, deleted, status change) or a title became available. */
    var requestsTick by mutableIntStateOf(0)
        private set
    var suggestionsTick by mutableIntStateOf(0)
        private set
    var issuesTick by mutableIntStateOf(0)
        private set
    var syncTick by mutableIntStateOf(0)
        private set

    /** Latest progress per request id; cleared when the request changes state. */
    val progress = mutableStateMapOf<String, LiveProgress>()

    private var hasConnectedOnce = false
    private var job: Job? = null
    private var runningFor: String? = null

    fun start(serverUrl: String, token: String) {
        val key = serverUrl + token
        if (runningFor == key) return
        stop()
        runningFor = key
        job = scope.launch {
            EventStream.run(
                serverUrl, token,
                onConnected = { withContext(Dispatchers.Main) { didConnect() } },
                onEvent = { withContext(Dispatchers.Main) { handle(it) } },
            )
        }
    }

    fun stop() {
        job?.cancel()
        job = null
        runningFor = null
    }

    /**
     * After every reconnection everything is refetched: events sent while away are not replayed to this client.
     * The very first connection does not, the screens just loaded their data themselves.
     */
    private fun didConnect() {
        val reconnect = hasConnectedOnce
        if (com.brebond.tipsarr.BuildConfig.DEBUG) android.util.Log.d("TipsarrLive", "connected (reconnect: $reconnect)")
        hasConnectedOnce = true
        if (!reconnect) return
        requestsTick++
        suggestionsTick++
        issuesTick++
        syncTick++
    }

    private fun handle(event: LiveEvent) {
        if (com.brebond.tipsarr.BuildConfig.DEBUG) android.util.Log.d("TipsarrLive", "${event.type} ${event.itemId.orEmpty()} ${event.percent?.let { "$it%" }.orEmpty()}")
        when (event.type) {
            "request.progress" -> {
                val id = event.itemId
                val percent = event.percent
                if (id != null && percent != null) progress[id] = LiveProgress(percent, event.etaSeconds ?: 0)
            }
            "request.updated" -> {
                event.itemId?.let { progress.remove(it) }
                requestsTick++
            }
            "media.available" -> requestsTick++
            "suggestions.updated" -> suggestionsTick++
            "issue.updated" -> issuesTick++
            "sync.status" -> syncTick++
        }
    }
}

val LocalLive = compositionLocalOf<LiveUpdates?> { null }

/** Runs `action` each time `tick` changes (not on first composition). */
@Composable
fun OnTick(tick: Int, action: suspend () -> Unit) {
    var seen by remember { mutableStateOf(tick) }
    LaunchedEffect(tick) {
        if (tick != seen) {
            seen = tick
            action()
        }
    }
}
