package com.brebond.tipsarr.core.api

import kotlinx.serialization.Serializable

/** Lean copies of what the app reads from the API (unknown fields are ignored). */
@Serializable
data class ServerStatus(
    val version: String,
    val apiVersion: Int,
    val minAppVersion: String,
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
    /** The stored mask and what the person may really do (everything for an admin); see [Permission]. */
    val permissions: Long = Permission.DEFAULT,
    val effectivePermissions: Long = Permission.DEFAULT,
    /** Request limits of their own (admin list only); absent = the default applies. */
    val movieQuotaLimit: Int? = null,
    val movieQuotaDays: Int? = null,
    val tvQuotaLimit: Int? = null,
    val tvQuotaDays: Int? = null,
) {
    /** The role: admins can do everything. For what a person may do, use [can]. */
    val isAdmin: Boolean get() = role == "admin"

    fun can(vararg flags: Long): Boolean = isAdmin || Permission.has(effectivePermissions, *flags)

    /** May request this kind of title. */
    fun mayRequest(type: MediaType): Boolean =
        can(Permission.REQUEST) || can(if (type == MediaType.Movie) Permission.REQUEST_MOVIE else Permission.REQUEST_TV)

    /** Requests of this kind are approved right away. */
    fun autoApproves(type: MediaType): Boolean =
        can(Permission.MANAGE_REQUESTS) || can(Permission.AUTO_APPROVE) || can(if (type == MediaType.Movie) Permission.AUTO_APPROVE_MOVIE else Permission.AUTO_APPROVE_TV)
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
