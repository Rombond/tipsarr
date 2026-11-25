package com.brebond.tipsarr.core.push

import android.Manifest
import android.annotation.SuppressLint
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.os.Build
import androidx.core.app.NotificationCompat
import androidx.core.app.NotificationManagerCompat
import androidx.core.content.ContextCompat
import com.brebond.tipsarr.MainActivity
import com.brebond.tipsarr.R
import com.brebond.tipsarr.features.detail.formatEta

/** The ongoing notification of a followed request: a Live Update on Android 16+, a progress notification before. */
object LiveFollowNotification {
    const val CHANNEL = "live"
    private const val FINAL_DISMISS_MS = 3_600_000L

    private fun tag(id: String) = "live:$id"

    fun createChannel(context: Context) {
        if (Build.VERSION.SDK_INT < 26) return
        val channel = NotificationChannel(CHANNEL, context.getString(R.string.m_follow_channel), NotificationManager.IMPORTANCE_DEFAULT).apply {
            setSound(null, null)
            enableVibration(false)
        }
        context.getSystemService(NotificationManager::class.java).createNotificationChannel(channel)
    }

    fun cancel(context: Context, id: String) = NotificationManagerCompat.from(context).cancel(tag(id), 0)

    @SuppressLint("MissingPermission") // checked just before
    fun post(context: Context, state: LiveState, entry: LiveFollow.Entry, final: Boolean = false) {
        if (Build.VERSION.SDK_INT >= 33 && ContextCompat.checkSelfPermission(context, Manifest.permission.POST_NOTIFICATIONS) != PackageManager.PERMISSION_GRANTED) return
        createChannel(context)
        val open = Intent(context, MainActivity::class.java)
            .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_SINGLE_TOP or Intent.FLAG_ACTIVITY_CLEAR_TOP)
            .putExtra("event", "request.live").putExtra("id", state.id).putExtra("server", entry.server)
        val tap = PendingIntent.getActivity(context, tag(state.id).hashCode(), open, PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE)
        val downloading = state.stage == "downloading"
        val text = buildString {
            append(stageLabel(context, state.stage))
            if (downloading && state.etaSeconds > 0) append(" · ").append(context.getString(R.string.m_follow_eta, formatEta(state.etaSeconds)))
        }
        val builder = NotificationCompat.Builder(context, CHANNEL)
            .setSmallIcon(R.drawable.ic_notification)
            .setContentTitle(entry.title)
            .setContentText(text)
            .setContentIntent(tap)
            .setOnlyAlertOnce(true)
            .setCategory(NotificationCompat.CATEGORY_PROGRESS)
            .setOngoing(!final)
            .setAutoCancel(final)
        if (final) {
            builder.setTimeoutAfter(if (state.dismissAfter > 0) state.dismissAfter * 1000 else FINAL_DISMISS_MS)
        } else {
            val left = entry.expiresAt * 1000 - System.currentTimeMillis()
            if (entry.expiresAt > 0 && left > 0) builder.setTimeoutAfter(left)
            val percent = if (state.stage == "available") 100 else state.percent
            if (Build.VERSION.SDK_INT >= 36) {
                builder.setStyle(
                    NotificationCompat.ProgressStyle()
                        .setProgressIndeterminate(!downloading)
                        .setProgress(percent),
                )
                builder.setRequestPromotedOngoing(true)
                if (downloading) builder.setShortCriticalText("$percent%")
            } else {
                builder.setProgress(100, percent, !downloading)
            }
        }
        NotificationManagerCompat.from(context).notify(tag(state.id), 0, builder.build())
    }

    private fun stageLabel(context: Context, stage: String): String {
        val id = context.resources.getIdentifier("state_$stage", "string", context.packageName)
        return if (id == 0) stage else context.getString(id)
    }
}
