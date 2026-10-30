package com.brebond.tipsarr.core.push

import android.Manifest
import android.app.PendingIntent
import android.content.Intent
import android.content.pm.PackageManager
import android.os.Build
import androidx.core.app.NotificationCompat
import androidx.core.app.NotificationManagerCompat
import androidx.core.content.ContextCompat
import com.brebond.tipsarr.MainActivity
import com.brebond.tipsarr.R
import com.google.firebase.messaging.FirebaseMessagingService
import com.google.firebase.messaging.RemoteMessage

/** Receives the FCM token and alerts. While the app is closed the system shows the alert itself; here it is shown while the app is open. */
class TipsarrMessagingService : FirebaseMessagingService() {
    override fun onNewToken(token: String) = PushManager.onNewToken(token)

    override fun onMessageReceived(message: RemoteMessage) {
        val event = message.data["event"] ?: return
        val id = message.data["id"] ?: return
        val title = message.notification?.title ?: return
        if (Build.VERSION.SDK_INT >= 33 && ContextCompat.checkSelfPermission(this, Manifest.permission.POST_NOTIFICATIONS) != PackageManager.PERMISSION_GRANTED) return
        PushManager.createChannels(this)
        val open = Intent(this, MainActivity::class.java)
            .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_SINGLE_TOP or Intent.FLAG_ACTIVITY_CLEAR_TOP)
            .putExtra("event", event).putExtra("id", id)
        message.data["server"]?.let { open.putExtra("server", it) }
        val tap = PendingIntent.getActivity(this, id.hashCode(), open, PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE)
        val notification = NotificationCompat.Builder(this, message.notification?.channelId ?: PushManager.channel(event))
            .setSmallIcon(R.drawable.ic_notification)
            .setContentTitle(title)
            .setContentText(message.notification?.body)
            .setAutoCancel(true)
            .setContentIntent(tap)
            .build()
        // Same tag as the system's own alert: a repeated alert for one request replaces the earlier one.
        NotificationManagerCompat.from(this).notify(id, 0, notification)
    }
}
