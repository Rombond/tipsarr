package com.brebond.tipsarr.core.api

import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.delay
import kotlinx.coroutines.suspendCancellableCoroutine
import kotlinx.serialization.KSerializer
import kotlinx.serialization.SerializationException
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import okhttp3.Call
import okhttp3.Callback
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import okhttp3.Response
import java.io.IOException
import java.util.concurrent.TimeUnit
import kotlin.coroutines.resume
import kotlin.coroutines.resumeWithException

/**
 * One server and one account's token. Adds the headers of the behaviour spec, 15 s timeouts, no cache,
 * retries GETs twice (1 s, 3 s) on network errors and 502/503, never retries writes, and maps every
 * failure to [ApiError].
 */
class ApiClient(
    serverUrl: String,
    private val token: String?,
    private val language: String,
    private val appVersion: String,
) {
    private val base = serverUrl.trimEnd('/') + "/api/v1"

    suspend fun <T> get(path: String, serializer: KSerializer<T>): T = decode(execute(request(path).get().build(), retry = true), serializer)

    suspend fun <B, T> post(path: String, body: B, bodySerializer: KSerializer<B>, serializer: KSerializer<T>): T {
        val payload = json.encodeToString(bodySerializer, body).toRequestBody(JSON)
        return decode(execute(request(path).post(payload).build(), retry = false), serializer)
    }

    /** POST without a response body that matters. */
    suspend fun postEmpty(path: String) {
        execute(request(path).post("".toRequestBody(JSON)).build(), retry = false)
    }

    /** POST with a JSON body and no response body that matters (204). */
    suspend fun <B> postUnit(path: String, body: B, bodySerializer: KSerializer<B>) {
        execute(request(path).post(json.encodeToString(bodySerializer, body).toRequestBody(JSON)).build(), retry = false)
    }

    /** POST without a body, returning JSON. */
    suspend fun <T> postForResult(path: String, serializer: KSerializer<T>): T =
        decode(execute(request(path).post("".toRequestBody(JSON)).build(), retry = false), serializer)

    suspend fun <B, T> patch(path: String, body: B, bodySerializer: KSerializer<B>, serializer: KSerializer<T>): T {
        val payload = json.encodeToString(bodySerializer, body).toRequestBody(JSON)
        return decode(execute(request(path).patch(payload).build(), retry = false), serializer)
    }

    /** Uploads a JPEG as the `file` part of a multipart form. */
    suspend fun postJpeg(path: String, jpeg: ByteArray) {
        val body = okhttp3.MultipartBody.Builder().setType(okhttp3.MultipartBody.FORM)
            .addFormDataPart("file", "avatar.jpg", jpeg.toRequestBody("image/jpeg".toMediaType()))
            .build()
        execute(request(path).post(body).build(), retry = false)
    }

    suspend fun delete(path: String) {
        execute(request(path).delete().build(), retry = false)
    }

    private fun request(path: String): Request.Builder = Request.Builder()
        .url(base + path)
        .header("Accept", "application/json")
        .header("X-Tipsarr-Language", language)
        .header("X-Tipsarr-App", "android/$appVersion")
        .apply { if (token != null) header("Authorization", "Bearer $token") }

    private fun <T> decode(text: String, serializer: KSerializer<T>): T = try {
        json.decodeFromString(serializer, text)
    } catch (e: SerializationException) {
        throw ApiError.Unexpected
    } catch (e: IllegalArgumentException) {
        throw ApiError.Unexpected
    }

    private suspend fun execute(request: Request, retry: Boolean): String {
        var attempt = 0
        while (true) {
            try {
                val response = http.newCall(request).await()
                response.use {
                    val code = it.code
                    val text = it.body.string()
                    if (code in 200..299) return text
                    if (retry && (code == 502 || code == 503) && attempt < DELAYS.size) {
                        delay(DELAYS[attempt++])
                        return@use
                    }
                    throw parseError(code, text)
                }
            } catch (e: IOException) {
                if (retry && attempt < DELAYS.size) {
                    delay(DELAYS[attempt++])
                    continue
                }
                throw ApiError.Unreachable
            }
        }
    }

    /** Server errors carry `errors[i].location == "code"` with the stable code in `value`. */
    private fun parseError(status: Int, text: String): ApiError {
        val root = runCatching { json.parseToJsonElement(text).jsonObject }.getOrNull() ?: return ApiError.Http(status, null, null)
        val errors = (root["errors"] as? JsonArray).orEmpty()
        val code = errors.firstNotNullOfOrNull { element ->
            val item = element as? JsonObject ?: return@firstNotNullOfOrNull null
            if (item["location"]?.jsonPrimitive?.contentOrNull == "code") item["value"]?.jsonPrimitive?.contentOrNull else null
        }
        return ApiError.Http(status, code, root["detail"]?.jsonPrimitive?.contentOrNull)
    }

    private suspend fun Call.await(): Response = suspendCancellableCoroutine { continuation ->
        enqueue(object : Callback {
            override fun onFailure(call: Call, e: IOException) { if (!continuation.isCancelled) continuation.resumeWithException(e) }
            override fun onResponse(call: Call, response: Response) { continuation.resume(response) }
        })
        continuation.invokeOnCancellation { cancel() }
    }

    companion object {
        private val JSON = "application/json".toMediaType()
        private val DELAYS = listOf(1_000L, 3_000L)
        val json = Json { ignoreUnknownKeys = true; explicitNulls = false }

        private val http = OkHttpClient.Builder()
            .connectTimeout(15, TimeUnit.SECONDS)
            .readTimeout(15, TimeUnit.SECONDS)
            .callTimeout(60, TimeUnit.SECONDS)
            .cache(null)
            .build()
    }
}
