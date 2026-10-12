package com.brebond.tipsarr.features.teasers

import android.content.Context
import android.view.LayoutInflater
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableFloatStateOf
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableLongStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberUpdatedState
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.viewinterop.AndroidView
import androidx.media3.common.AudioAttributes
import androidx.media3.common.C
import androidx.media3.common.MediaItem
import androidx.media3.common.PlaybackException
import androidx.media3.common.Player
import androidx.media3.common.util.UnstableApi
import androidx.media3.datasource.DefaultHttpDataSource
import androidx.media3.exoplayer.ExoPlayer
import androidx.media3.exoplayer.SeekParameters
import androidx.media3.exoplayer.source.DefaultMediaSourceFactory
import androidx.media3.ui.PlayerView
import com.brebond.tipsarr.R

/**
 * One trailer file played natively (ExoPlayer), fetched from the Tipsarr server with the bearer token.
 * Prepared as soon as it is created (the card before it is on screen), so the next swipe starts at once.
 * Sound follows the media volume (USAGE_MEDIA, movie content, audio focus). Loops; [loops] counts the completed ones.
 */
@OptIn(UnstableApi::class)
class TeaserVideo(context: Context, url: String, token: String?, private val onFailed: () -> Unit) {
    val player: ExoPlayer

    /** 0..1 through the trailer. */
    var progress by mutableFloatStateOf(0f)
        private set
    var positionMs by mutableLongStateOf(0L)
        private set
    var durationMs by mutableLongStateOf(0L)
        private set

    /** The first frame is on screen. */
    var ready by mutableStateOf(false)
        private set

    /** How many times the trailer played to its end. */
    var loops by mutableIntStateOf(0)
        private set

    /** A finger is on the progress bar: the clock leaves the playhead alone and playback is held. */
    var scrubbing by mutableStateOf(false)
        private set

    private var wantPlaying = false

    init {
        val http = DefaultHttpDataSource.Factory().apply {
            if (token != null) setDefaultRequestProperties(mapOf("Authorization" to "Bearer $token"))
        }
        val media = AudioAttributes.Builder().setUsage(C.USAGE_MEDIA).setContentType(C.AUDIO_CONTENT_TYPE_MOVIE).build()
        player = ExoPlayer.Builder(context)
            .setMediaSourceFactory(DefaultMediaSourceFactory(context).setDataSourceFactory(http))
            .setAudioAttributes(media, true)
            .setHandleAudioBecomingNoisy(true)
            .build()
            .apply {
                repeatMode = Player.REPEAT_MODE_ONE
                setMediaItem(MediaItem.fromUri(url))
                prepare()
            }
        player.addListener(object : Player.Listener {
            override fun onPlayerError(error: PlaybackException) = onFailed()
            override fun onRenderedFirstFrame() { ready = true }
            override fun onMediaItemTransition(mediaItem: MediaItem?, reason: Int) {
                if (reason == Player.MEDIA_ITEM_TRANSITION_REASON_REPEAT) loops++
            }
        })
    }

    private fun apply() { player.playWhenReady = wantPlaying && !scrubbing }

    /** Plays or pauses (a paused video keeps buffering). */
    fun setPlaying(playing: Boolean) { wantPlaying = playing; apply() }

    fun setMuted(muted: Boolean) { player.volume = if (muted) 0f else 1f }

    /** Hold-to-speed-up. */
    fun setFast(on: Boolean) = player.setPlaybackSpeed(if (on) 2f else 1f)

    /** Refreshes [progress] from the player (called a few times a second). */
    fun tick() {
        if (scrubbing) return
        val d = player.duration
        if (d <= 0) return
        durationMs = d
        positionMs = player.currentPosition.coerceIn(0L, d)
        progress = positionMs.toFloat() / d
    }

    fun beginScrub() {
        scrubbing = true
        player.setSeekParameters(SeekParameters.EXACT)
        apply()
    }

    /** Moves the playhead to this fraction (0..1) of the trailer. */
    fun scrub(fraction: Float) {
        val d = player.duration
        if (d <= 0) return
        val f = fraction.coerceIn(0f, 1f)
        durationMs = d
        progress = f
        positionMs = (f * d).toLong()
        player.seekTo(positionMs)
    }

    fun endScrub() {
        scrubbing = false
        apply()
        player.setSeekParameters(SeekParameters.DEFAULT)
    }

    fun release() = player.release()
}

/** A [TeaserVideo] for this file: created with the composable, released when it leaves the composition. */
@Composable
fun rememberTeaserVideo(url: String, token: String?, onFailed: () -> Unit): TeaserVideo {
    val context = LocalContext.current
    val failed by rememberUpdatedState(onFailed)
    val video = remember(url, token) { TeaserVideo(context.applicationContext, url, token) { failed() } }
    DisposableEffect(video) { onDispose { video.release() } }
    return video
}

/** The picture of a [TeaserVideo]: a TextureView (follows the pager's scrolling), the whole picture visible, no controls. */
@OptIn(UnstableApi::class)
@Composable
fun TeaserSurface(video: TeaserVideo, modifier: Modifier = Modifier) {
    AndroidView(
        factory = { ctx -> (LayoutInflater.from(ctx).inflate(R.layout.teaser_player_view, null) as PlayerView).also { it.player = video.player } },
        // The view follows the current player (a view that came back may hold an old one).
        update = { if (it.player !== video.player) it.player = video.player },
        onRelease = { it.player = null },
        modifier = modifier,
    )
}
