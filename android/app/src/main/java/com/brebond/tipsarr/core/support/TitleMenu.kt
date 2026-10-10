package com.brebond.tipsarr.core.support

import androidx.compose.runtime.compositionLocalOf
import com.brebond.tipsarr.core.api.ApiError
import com.brebond.tipsarr.core.api.MediaType
import com.brebond.tipsarr.core.api.TipsarrApi
import com.brebond.tipsarr.core.api.createRequest
import com.brebond.tipsarr.core.api.setBlocklisted
import com.brebond.tipsarr.core.api.setWatchlisted

/** What the long-press menu on a poster does: the title's defaults (every season, default folder), the result shown as a toast. */
class TitleMenu(private val api: TipsarrApi) {
    suspend fun request(type: MediaType, tmdbId: Int) { api.createRequest(type, tmdbId, null, null, null) }
    suspend fun watchlist(type: MediaType, tmdbId: Int) = api.setWatchlisted(true, type, tmdbId)
    suspend fun hide(type: MediaType, tmdbId: Int) = api.setBlocklisted(true, type, tmdbId)
}

val LocalTitleMenu = compositionLocalOf<TitleMenu?> { null }
