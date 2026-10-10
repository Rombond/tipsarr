package com.brebond.tipsarr.core.auth

import kotlinx.serialization.Serializable
import java.net.URI

/** One signed-in user on one server. Stored encrypted (see [SecureStore]). */
@Serializable
data class Account(
    val serverUrl: String,
    val userId: String,
    val name: String,
    val token: String,
    val lastUsed: Long,
) {
    val id: String get() = makeId(serverUrl, userId)
    val host: String get() = runCatching { URI(serverUrl).host }.getOrNull()?.lowercase() ?: serverUrl

    companion object {
        fun makeId(server: String, userId: String): String =
            "${runCatching { URI(server).host }.getOrNull() ?: server}|$userId"
    }
}
