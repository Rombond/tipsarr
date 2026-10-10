package com.brebond.tipsarr.core.api

import com.brebond.tipsarr.BuildConfig
import kotlinx.coroutines.CancellationException
import kotlinx.serialization.KSerializer
import java.util.Locale

/** The calls the launch flow needs. Throws [ApiError] only. */
class TipsarrApi(val serverUrl: String, val token: String? = null) {
    private val client = ApiClient(serverUrl, token, currentLanguage(), BuildConfig.APP_VERSION)

    suspend fun status(): ServerStatus = wrap { client.get("/status", ServerStatus.serializer()) }

    suspend fun signIn(username: String, password: String, deviceName: String): SignInResult = wrap {
        val input = TokenInput(username, password, "android", deviceName, BuildConfig.APP_VERSION)
        val body = client.post("/auth/token", input, TokenInput.serializer(), TokenBody.serializer())
        SignInResult(body.token, body.user)
    }

    suspend fun <B, T> post(path: String, body: B, bodySerializer: KSerializer<B>, serializer: KSerializer<T>): T =
        wrap { client.post(path, body, bodySerializer, serializer) }

    suspend fun <B> postUnit(path: String, body: B, bodySerializer: KSerializer<B>) = wrap { client.postUnit(path, body, bodySerializer) }

    suspend fun <T> postForResult(path: String, serializer: KSerializer<T>): T = wrap { client.postForResult(path, serializer) }

    suspend fun delete(path: String) = wrap { client.delete(path) }

    suspend fun postEmpty(path: String) = wrap { client.postEmpty(path) }

    suspend fun <B, T> patch(path: String, body: B, bodySerializer: KSerializer<B>, serializer: KSerializer<T>): T =
        wrap { client.patch(path, body, bodySerializer, serializer) }

    suspend fun postJpeg(path: String, jpeg: ByteArray) = wrap { client.postJpeg(path, jpeg) }

    /** One GET of a JSON resource, for the endpoint files (`TipsarrApi+Discover.kt`...). */
    suspend fun <T> get(path: String, serializer: KSerializer<T>): T = wrap { client.get(path, serializer) }

    suspend fun me(): Profile = wrap { client.get("/me", Profile.serializer()) }

    /** Best effort: the account is removed from the phone even when this fails. */
    suspend fun logout() {
        try { client.postEmpty("/auth/logout") } catch (_: ApiError) {}
    }

    private suspend fun <T> wrap(call: suspend () -> T): T = try {
        call()
    } catch (e: CancellationException) {
        throw e
    } catch (e: Throwable) {
        throw ApiError.from(e)
    }

    companion object {
        /** e.g. `fr-FR`; the profile language, when set, wins on the server. */
        fun currentLanguage(): String = Locale.getDefault().toLanguageTag()
    }
}
