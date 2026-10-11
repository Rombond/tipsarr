package com.brebond.tipsarr.features.teasers

import android.view.LayoutInflater
import androidx.compose.foundation.background
import androidx.compose.foundation.gestures.detectTapGestures
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.size
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.PlayCircle
import androidx.compose.material3.Icon
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableFloatStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import androidx.compose.ui.viewinterop.AndroidView
import androidx.media3.common.MediaItem
import androidx.media3.common.PlaybackException
import androidx.media3.common.Player
import androidx.media3.common.util.UnstableApi
import androidx.media3.datasource.DefaultHttpDataSource
import androidx.media3.exoplayer.ExoPlayer
import androidx.media3.exoplayer.source.DefaultMediaSourceFactory
import androidx.media3.ui.PlayerView
import com.brebond.tipsarr.R
import kotlinx.coroutines.delay

/**
 * One trailer file played natively (ExoPlayer). The file is fetched from the Tipsarr server with the bearer token.
 * It is prepared as soon as the composable exists (the card before it is on screen), so the next swipe starts at once,
 * and plays only while [active]. Tap pauses or resumes; loops; volume follows [muted]. [onFailed] fires on any playback error.
 */
@OptIn(UnstableApi::class)
@Composable
fun FilePlayer(url: String, token: String?, active: Boolean, muted: Boolean, modifier: Modifier, onFailed: () -> Unit) {
    val context = LocalContext.current
    var userPaused by remember(url) { mutableStateOf(false) }
    var progress by remember(url) { mutableFloatStateOf(0f) }
    val player = remember(url) {
        val http = DefaultHttpDataSource.Factory().apply {
            if (token != null) setDefaultRequestProperties(mapOf("Authorization" to "Bearer $token"))
        }
        ExoPlayer.Builder(context).setMediaSourceFactory(DefaultMediaSourceFactory(context).setDataSourceFactory(http)).build().apply {
            repeatMode = Player.REPEAT_MODE_ONE
            setMediaItem(MediaItem.fromUri(url))
            prepare()
        }
    }
    DisposableEffect(player) {
        val listener = object : Player.Listener {
            override fun onPlayerError(error: PlaybackException) = onFailed()
        }
        player.addListener(listener)
        onDispose { player.removeListener(listener); player.release() }
    }
    LaunchedEffect(player, muted) { player.volume = if (muted) 0f else 1f }
    LaunchedEffect(player, active, userPaused) { player.playWhenReady = active && !userPaused }
    LaunchedEffect(active) { if (!active) userPaused = false } // a card starts playing again when it comes back
    LaunchedEffect(player, active) {
        while (active) {
            val duration = player.duration
            progress = if (duration > 0) (player.currentPosition.toFloat() / duration).coerceIn(0f, 1f) else 0f
            delay(250)
        }
    }

    Box(modifier.background(Color.Black)) {
        AndroidView(
            factory = { ctx -> (LayoutInflater.from(ctx).inflate(R.layout.teaser_player_view, null) as PlayerView).also { it.player = player } },
            update = { it.player = player },
            modifier = Modifier.fillMaxSize(),
        )
        // Tap to pause or resume; vertical swipes still reach the pager.
        Box(Modifier.fillMaxSize().pointerInput(Unit) { detectTapGestures(onTap = { userPaused = !userPaused }) })
        if (userPaused) {
            Icon(Icons.Outlined.PlayCircle, contentDescription = null, tint = Color.White.copy(alpha = 0.85f), modifier = Modifier.align(Alignment.Center).size(56.dp))
        }
        LinearProgressIndicator(
            progress = { progress },
            modifier = Modifier.align(Alignment.BottomCenter).fillMaxWidth().height(2.dp),
            color = Color.White,
            trackColor = Color.White.copy(alpha = 0.25f),
            drawStopIndicator = {},
        )
    }
}
