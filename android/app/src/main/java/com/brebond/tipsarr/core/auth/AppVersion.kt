package com.brebond.tipsarr.core.auth

import java.net.URI

object AppVersion {
    /** True when [current] is older than [minimum]. An empty minimum means any version. */
    fun isOlder(current: String, minimum: String): Boolean {
        if (minimum.isEmpty()) return false
        val a = parts(current)
        val b = parts(minimum)
        for (i in 0 until maxOf(a.size, b.size)) {
            val x = a.getOrElse(i) { 0 }
            val y = b.getOrElse(i) { 0 }
            if (x != y) return x < y
        }
        return false
    }

    private fun parts(version: String): List<Int> =
        version.split('.').map { part -> part.takeWhile { it.isDigit() }.toIntOrNull() ?: 0 }
}

object ServerAddress {
    /** "tipsarr.example.com/" becomes https://tipsarr.example.com. Null when it cannot be a server URL. */
    fun normalize(raw: String): String? {
        var text = raw.trim()
        if (text.isEmpty()) return null
        if (!text.contains("://")) text = "https://$text"
        val uri = runCatching { URI(text) }.getOrNull() ?: return null
        val scheme = uri.scheme?.lowercase() ?: return null
        val host = uri.host?.takeIf { it.isNotEmpty() } ?: return null
        if (scheme != "http" && scheme != "https") return null
        val port = if (uri.port >= 0) ":${uri.port}" else ""
        return "$scheme://$host$port${uri.rawPath.orEmpty().trimEnd('/')}"
    }
}
