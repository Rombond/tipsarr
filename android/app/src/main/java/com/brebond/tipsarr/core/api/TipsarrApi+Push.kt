package com.brebond.tipsarr.core.api

import kotlinx.serialization.Serializable

/** What the server keeps for this install (never the token itself). */
data class PushRegistration(val categories: Int, val language: String)

@Serializable
private data class RegisterDeviceBody(
    val pushToken: String,
    val language: String,
    val server: String,
    val categories: Long? = null,
)

@Serializable
private data class PushDeviceBody(val categories: Long = 7, val language: String = "en")

/** `server` is the host (and port) this app reaches the server on; it comes back in every alert. `categories` null keeps what the server has. */
suspend fun TipsarrApi.registerDevice(token: String, language: String, server: String, categories: Int?): PushRegistration {
    val device = put(
        "/me/devices/current", RegisterDeviceBody(token, language, server, categories?.toLong()),
        RegisterDeviceBody.serializer(), PushDeviceBody.serializer(),
    )
    return PushRegistration(device.categories.toInt(), device.language)
}

suspend fun TipsarrApi.unregisterDevice() = delete("/me/devices/current")
