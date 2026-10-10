package com.brebond.tipsarr.core.live

import com.brebond.tipsarr.BuildConfig
import com.brebond.tipsarr.core.api.ApiClient
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.currentCoroutineContext
import kotlinx.coroutines.delay
import kotlinx.coroutines.ensureActive
import kotlinx.coroutines.isActive
import kotlinx.coroutines.withContext
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.intOrNull
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import okhttp3.OkHttpClient
import okhttp3.Request
import java.io.IOException
import java.util.concurrent.TimeUnit

/**
 * One server-sent event: `request.updated`, `request.progress`, `media.available`, `suggestions.updated`,
 * `issue.updated`, `sync.status`. Events are nudges; the data only identifies what changed.
 */
data class LiveEvent(val id: Int?, val type: String, val itemId: String?, val percent: Int?, val etaSeconds: Int?)

/** Streams `GET /api/v1/events` (SSE) and reconnects with `Last-Event-ID`. Cancel the coroutine to stop it. */
object EventStream {
    // The server pings every 20 s, so a 90 s read timeout means the connection is dead.
    private val http = OkHttpClient.Builder()
        .connectTimeout(15, TimeUnit.SECONDS)
        .readTimeout(90, TimeUnit.SECONDS)
        .callTimeout(0, TimeUnit.SECONDS)
        .build()

    /** `onEvent` runs for every event; `onConnected` runs after every (re)connection so callers can refetch. */
    suspend fun run(serverUrl: String, token: String, onConnected: suspend () -> Unit, onEvent: suspend (LiveEvent) -> Unit) {
        var lastId: Int? = null
        var delayMs = 1_000L
        while (currentCoroutineContext().isActive) {
            val request = Request.Builder()
                .url(serverUrl.trimEnd('/') + "/api/v1/events")
                .header("Authorization", "Bearer $token")
                .header("Accept", "text/event-stream")
                .header("X-Tipsarr-App", "android/${BuildConfig.APP_VERSION}")
                .apply { lastId?.let { header("Last-Event-ID", it.toString()) } }
                .build()
            val call = http.newCall(request)
            val job = currentCoroutineContext()[kotlinx.coroutines.Job]
            val handle = job?.invokeOnCompletion { call.cancel() }
            try {
                withContext(Dispatchers.IO) { call.execute() }.use { response ->
                    if (response.code != 200) throw IOException("HTTP ${response.code}")
                    delayMs = 1_000L
                    onConnected()
                    val source = response.body.source()
                    var id: Int? = null
                    var type: String? = null
                    val data = StringBuilder()
                    while (true) {
                        currentCoroutineContext().ensureActive()
                        val line = withContext(Dispatchers.IO) { source.readUtf8Line() } ?: break
                        when {
                            line.isEmpty() -> {
                                // A blank line ends an event; comments (": ping") and the retry hint give none.
                                val kind = type
                                if (kind != null) {
                                    val event = parse(id, kind, data.toString())
                                    lastId = event.id ?: lastId
                                    onEvent(event)
                                }
                                id = null; type = null; data.clear()
                            }
                            line.startsWith("id:") -> id = line.removePrefix("id:").trim().toIntOrNull()
                            line.startsWith("event:") -> type = line.removePrefix("event:").trim()
                            line.startsWith("data:") -> data.append(line.removePrefix("data:").trim())
                        }
                    }
                }
            } catch (e: IOException) {
                // Dropped or refused: wait, then try again.
            } finally {
                handle?.dispose()
            }
            if (!currentCoroutineContext().isActive) return
            // 1 s at first, 30 s at most.
            delay(delayMs)
            delayMs = minOf(delayMs * 2, 30_000L)
        }
    }

    internal fun parse(id: Int?, type: String, data: String): LiveEvent {
        val obj: JsonObject? = runCatching { ApiClient.json.parseToJsonElement(data).jsonObject }.getOrNull()
        return LiveEvent(
            id = id, type = type,
            itemId = obj?.get("id")?.jsonPrimitive?.contentOrNull,
            percent = obj?.get("percent")?.jsonPrimitive?.intOrNull,
            etaSeconds = obj?.get("etaSeconds")?.jsonPrimitive?.intOrNull,
        )
    }
}
