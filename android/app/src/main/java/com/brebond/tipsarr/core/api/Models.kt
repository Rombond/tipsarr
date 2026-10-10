package com.brebond.tipsarr.core.api

import kotlinx.serialization.Serializable

/** Lean copies of what the app reads from the API (unknown fields are ignored). */
@Serializable
data class ServerStatus(
    val version: String,
    val apiVersion: Int,
    val minAppVersion: String,
    val userFolderChoice: Boolean,
    val defaultLanguage: String,
    val features: Features,
) {
    @Serializable
    data class Features(val push: Boolean)
}

@Serializable
data class Profile(
    val id: String,
    val name: String,
    val role: String,
    val region: String,
    val language: String,
    val ratingSource: String,
    val createdAt: Long = 0,
    val lastLoginAt: Long = 0,
) {
    val isAdmin: Boolean get() = role == "admin"
}

@Serializable
data class TokenInput(val username: String, val password: String, val platform: String, val deviceName: String, val appVersion: String)

@Serializable
data class TokenBody(val token: String, val expiresAt: Long, val user: Profile)

data class SignInResult(val token: String, val profile: Profile)

/** A server the user connected to, with the status read at that moment. */
data class Server(val url: String, val status: ServerStatus) {
    val host: String get() = runCatching { java.net.URI(url).host }.getOrNull() ?: url
}
