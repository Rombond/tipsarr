package com.brebond.tipsarr.core.navigation

import com.brebond.tipsarr.core.api.MediaType

sealed interface DeepLinkTarget {
    data class Media(val type: MediaType, val tmdbId: Int) : DeepLinkTarget
    data class Request(val id: String) : DeepLinkTarget
    data class Issue(val id: String) : DeepLinkTarget
    data object Library : DeepLinkTarget
    data object Requests : DeepLinkTarget
    data object Discover : DeepLinkTarget
}

/**
 * `tipsarr://<server-host>/media/{movie|tv}/{tmdbId}`, `/request/{id}`, `/issue/{id}`, `/library`, `/requests`.
 * Unknown paths open Discover. Same links as the iOS app.
 */
data class DeepLink(val host: String, val target: DeepLinkTarget) {
    companion object {
        fun parse(scheme: String?, host: String?, segments: List<String>): DeepLink? {
            if (scheme?.lowercase() != "tipsarr" || host.isNullOrEmpty()) return null
            val parts = segments.filter { it.isNotEmpty() && it != "/" }
            val target: DeepLinkTarget = when (parts.firstOrNull()) {
                "media" -> {
                    val type = when (parts.getOrNull(1)) { "movie" -> MediaType.Movie; "tv" -> MediaType.Tv; else -> null }
                    val id = parts.getOrNull(2)?.toIntOrNull()
                    if (parts.size == 3 && type != null && id != null) DeepLinkTarget.Media(type, id) else DeepLinkTarget.Discover
                }
                "request" -> if (parts.size == 2) DeepLinkTarget.Request(parts[1]) else DeepLinkTarget.Discover
                "issue" -> if (parts.size == 2) DeepLinkTarget.Issue(parts[1]) else DeepLinkTarget.Discover
                "library" -> DeepLinkTarget.Library
                "requests" -> DeepLinkTarget.Requests
                else -> DeepLinkTarget.Discover
            }
            return DeepLink(host.lowercase(), target)
        }

        fun parse(uri: android.net.Uri): DeepLink? = parse(uri.scheme, uri.host, uri.pathSegments.orEmpty())

        /** Link that opens a title in the app of someone signed in to the same server. */
        fun mediaUrl(host: String, type: MediaType, tmdbId: Int): String =
            "tipsarr://$host/media/${if (type == MediaType.Movie) "movie" else "tv"}/$tmdbId"
    }
}
