package com.brebond.tipsarr.features.requests

import android.content.Context
import com.brebond.tipsarr.R
import com.brebond.tipsarr.core.api.ApiError
import com.brebond.tipsarr.core.api.MediaType
import com.brebond.tipsarr.core.api.RequestRecord
import com.brebond.tipsarr.core.push.LiveFollow
import com.brebond.tipsarr.core.support.ToastCenter
import com.brebond.tipsarr.core.support.ToastKind

/** Follow (or stop following) a request outside the app, with the toast that tells how it went. */
internal suspend fun toggleFollow(context: Context, toast: ToastCenter?, record: RequestRecord, following: Boolean) {
    if (following) { LiveFollow.unfollow(record.id); return }
    try {
        val type = if (record.type == MediaType.Tv) "tv" else "movie"
        when (LiveFollow.follow(record.id, record.title, type)) {
            LiveFollow.Result.Started -> toast?.show(context.getString(R.string.m_requests_follow_started, record.title))
            LiveFollow.Result.NotificationsOff -> toast?.show(context.getString(R.string.m_requests_follow_notif_off), ToastKind.Error)
            null -> {}
        }
    } catch (e: ApiError) {
        toast?.show(e.message(context), ToastKind.Error)
    }
}

/** A request can be followed while it is not finished. */
internal val RequestRecord.canFollow: Boolean get() = isOpen
