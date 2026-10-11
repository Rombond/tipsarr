package com.brebond.tipsarr.features.teasers

import android.annotation.SuppressLint
import android.os.Handler
import android.os.Looper
import android.webkit.JavascriptInterface
import android.webkit.WebChromeClient
import android.webkit.WebView
import androidx.compose.animation.AnimatedVisibility
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.pager.VerticalPager
import androidx.compose.foundation.pager.rememberPagerState
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Bookmark
import androidx.compose.material.icons.outlined.BookmarkBorder
import androidx.compose.material.icons.outlined.Info
import androidx.compose.material.icons.outlined.PlayCircle
import androidx.compose.material.icons.outlined.VisibilityOff
import androidx.compose.material.icons.outlined.VolumeOff
import androidx.compose.material.icons.outlined.VolumeUp
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.runtime.snapshotFlow
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.viewinterop.AndroidView
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.LifecycleEventObserver
import androidx.lifecycle.compose.LocalLifecycleOwner
import com.brebond.tipsarr.R
import com.brebond.tipsarr.core.api.ApiError
import com.brebond.tipsarr.core.api.Availability
import com.brebond.tipsarr.core.api.MediaItem
import com.brebond.tipsarr.core.api.MediaType
import com.brebond.tipsarr.core.api.Profile
import com.brebond.tipsarr.core.api.Teaser
import com.brebond.tipsarr.core.api.TipsarrApi
import com.brebond.tipsarr.core.images.RemoteImage
import com.brebond.tipsarr.core.images.TmdbSize
import com.brebond.tipsarr.core.support.AppSettings
import com.brebond.tipsarr.core.support.LocalToast
import com.brebond.tipsarr.core.support.ToastKind
import com.brebond.tipsarr.core.support.stringByName
import com.brebond.tipsarr.design.Tokens
import com.brebond.tipsarr.design.palette
import com.brebond.tipsarr.features.detail.MediaDetailModel
import com.brebond.tipsarr.features.detail.MediaRoute
import com.brebond.tipsarr.features.detail.RequestSheet
import com.brebond.tipsarr.features.discover.Phase
import com.brebond.tipsarr.ui.ButtonKind
import com.brebond.tipsarr.ui.RequestState
import com.brebond.tipsarr.ui.TipsarrButton
import kotlinx.coroutines.launch

/**
 * The Teasers tab: one trailer per screen, swipe up for the next. The video plays with sound
 * (through the phone's media volume) while its card is settled on screen; the mute button is remembered.
 */
@Composable
fun TeasersScreen(
    model: TeasersModel,
    api: TipsarrApi,
    me: () -> Profile,
    settings: AppSettings,
    onOpenItem: (MediaItem) -> Unit,
    modifier: Modifier = Modifier,
) {
    val scope = rememberCoroutineScope()
    LaunchedEffect(model) { model.loadIfNeeded() }
    // A narrow 9:16 column in the middle on wide windows, the whole width on a phone.
    BoxWithConstraints(modifier.fillMaxSize().background(Color.Black), contentAlignment = Alignment.Center) {
        val column = minOf(maxWidth, maxHeight * 9f / 16f)
        Box(Modifier.width(column).fillMaxHeight()) {
            when (model.phase) {
                Phase.Idle, Phase.Loading -> Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) { CircularProgressIndicator(color = Color.White) }
                is Phase.Failed -> DarkState(stringResource(R.string.m_teasers_error), stringResource(R.string.common_retry)) { scope.launch { model.refresh() } }
                Phase.Loaded -> Feed(model, api, me, settings, onOpenItem)
            }
        }
    }
}

/** Message on the black background (the app's state view is themed for light screens). */
@Composable
private fun DarkState(text: String, actionTitle: String? = null, onAction: () -> Unit = {}) {
    Column(
        Modifier.fillMaxSize().background(Color.Black).padding(Tokens.Spacing.x3xl),
        verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.lg, Alignment.CenterVertically),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Icon(Icons.Outlined.PlayCircle, contentDescription = null, tint = Color.White.copy(alpha = 0.7f), modifier = Modifier.size(40.dp))
        Text(text, color = Color.White, fontSize = Tokens.FontSize.headline, fontWeight = FontWeight.SemiBold, textAlign = TextAlign.Center)
        if (actionTitle != null) TipsarrButton(actionTitle, onAction)
    }
}

@Composable
private fun Feed(model: TeasersModel, api: TipsarrApi, me: () -> Profile, settings: AppSettings, onOpenItem: (MediaItem) -> Unit) {
    val context = LocalContext.current
    val toast = LocalToast.current
    val scope = rememberCoroutineScope()
    val count = model.items.size + if (model.done) 1 else 0
    val pager = rememberPagerState(initialPage = model.page.coerceIn(0, (count - 1).coerceAtLeast(0))) { count }
    val settled = pager.settledPage

    // The card on screen is remembered; the ones passed are reported as seen.
    var last by remember { mutableStateOf(settled) }
    var hintGone by rememberSaveable { mutableStateOf(false) }
    LaunchedEffect(Unit) {
        snapshotFlow { pager.settledPage }.collect { now ->
            model.page = now
            if (now > last) {
                hintGone = true
                for (i in last until now) model.items.getOrNull(i)?.let { launch { model.markSeen(it) } }
            }
            last = now
        }
    }
    // Three cards before the end, the next batch is fetched.
    LaunchedEffect(settled, model.items.size) {
        if (settled >= model.items.size - 3) model.loadMore()
    }

    // Playback follows the app: paused in the background.
    val owner = LocalLifecycleOwner.current
    var resumed by remember { mutableStateOf(owner.lifecycle.currentState.isAtLeast(Lifecycle.State.RESUMED)) }
    DisposableEffect(owner) {
        val observer = LifecycleEventObserver { _, event ->
            if (event == Lifecycle.Event.ON_PAUSE) resumed = false else if (event == Lifecycle.Event.ON_RESUME) resumed = true
        }
        owner.lifecycle.addObserver(observer)
        onDispose { owner.lifecycle.removeObserver(observer) }
    }

    // Request from a card: the same flow as the title screen (seasons, quality, quota).
    var sheet by remember { mutableStateOf<Pair<MediaDetailModel, Teaser>?>(null) }
    var requesting by remember { mutableStateOf(false) }
    fun announce(title: String, record: com.brebond.tipsarr.core.api.RequestRecord) {
        val key = if (record.state == RequestState.Requested) "media_toast_requested" else if (record.dryRun) "media_toast_approved_dry" else "media_toast_approved"
        toast?.show((context.stringByName(key) ?: "").replace("%1\$s", title).replace("%s", title))
    }
    fun request(teaser: Teaser) {
        if (requesting) return
        scope.launch {
            requesting = true
            try {
                val detail = MediaDetailModel(MediaRoute(teaser.type, teaser.tmdbId, teaser.title), api, me)
                detail.load()
                if (detail.prepareRequest()) {
                    sheet = detail to teaser
                } else {
                    val seasons = if (teaser.type == MediaType.Tv) detail.regularSeasons.map { it.number } else null
                    val record = detail.submitRequest(seasons, null, null)
                    announce(teaser.title, record)
                    model.requested(teaser, record.state != RequestState.Requested)
                }
            } catch (e: ApiError) {
                toast?.show(e.message(context), ToastKind.Error)
            } finally {
                requesting = false
            }
        }
    }
    fun guarded(work: suspend () -> Unit) {
        scope.launch { try { work() } catch (e: ApiError) { toast?.show(e.message(context), ToastKind.Error) } }
    }

    VerticalPager(pager, Modifier.fillMaxSize(), key = { index -> model.items.getOrNull(index)?.id ?: "end" }) { index ->
        val teaser = model.items.getOrNull(index)
        if (teaser == null) {
            DarkState(stringResource(R.string.m_teasers_done))
        } else {
            TeaserCard(
                teaser = teaser,
                active = index == settled && resumed,
                muted = settings.teaserMuted,
                showHint = index == 0 && !hintGone,
                mayRequest = me().mayRequest(teaser.type),
                requesting = requesting,
                onToggleMute = { settings.updateTeaserMuted(!settings.teaserMuted) },
                onRequest = { request(teaser) },
                onDetails = { onOpenItem(teaser.item) },
                onHide = { guarded { model.hide(teaser) } },
                onWatchlist = { guarded { model.toggleWatchlist(teaser) } },
                onUnplayable = { scope.launch { model.skip(teaser) } },
            )
        }
    }

    sheet?.let { (detail, teaser) ->
        RequestSheet(detail, onDismiss = { sheet = null }) { record ->
            sheet = null
            announce(teaser.title, record)
            model.requested(teaser, record.state != RequestState.Requested)
        }
    }
}

@Composable
private fun TeaserCard(
    teaser: Teaser,
    active: Boolean,
    muted: Boolean,
    showHint: Boolean,
    mayRequest: Boolean,
    requesting: Boolean,
    onToggleMute: () -> Unit,
    onRequest: () -> Unit,
    onDetails: () -> Unit,
    onHide: () -> Unit,
    onWatchlist: () -> Unit,
    onUnplayable: () -> Unit,
) {
    var playing by remember(teaser.id) { mutableStateOf(false) }
    Box(Modifier.fillMaxSize().background(Color.Black)) {
        // The backdrop shows until the video runs (and stays behind it).
        RemoteImage(teaser.backdropPath ?: teaser.posterPath, TmdbSize.W780, Modifier.fillMaxSize().alpha(if (playing) 0f else 0.55f))
        if (active) {
            YouTubePlayer(
                teaser.trailerKey, muted, Modifier.fillMaxSize(),
                onPlaying = { playing = it },
                onUnplayable = onUnplayable,
            )
            // The WebView would swallow the swipe; this layer lets the pager see it.
            Box(Modifier.fillMaxSize().pointerInput(Unit) {})
        }
        Box(Modifier.fillMaxWidth().fillMaxHeight(0.45f).align(Alignment.BottomCenter).background(Brush.verticalGradient(listOf(Color.Transparent, Color.Black.copy(alpha = 0.85f)))))

        // Title, year, rating, overview and the request button.
        Column(
            Modifier.align(Alignment.BottomStart).navigationBarsPadding().fillMaxWidth(0.78f).padding(start = Tokens.Spacing.lg, bottom = Tokens.Spacing.xl),
            verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm),
        ) {
            Text(teaser.title, color = Color.White, fontSize = Tokens.FontSize.title3, fontWeight = FontWeight.Bold, maxLines = 2, overflow = TextOverflow.Ellipsis)
            val meta = listOfNotNull(
                teaser.releaseDate?.take(4)?.takeIf { it.isNotEmpty() },
                teaser.voteAverage.takeIf { it > 0 }?.let { "★ %.1f".format(it) },
            ).joinToString(" · ")
            if (meta.isNotEmpty()) Text(meta, color = Color.White.copy(alpha = 0.8f), fontSize = Tokens.FontSize.footnote)
            teaser.overview?.takeIf { it.isNotBlank() }?.let { Text(it, color = Color.White.copy(alpha = 0.9f), fontSize = Tokens.FontSize.footnote, maxLines = 2, overflow = TextOverflow.Ellipsis) }
            RequestButton(teaser, mayRequest, requesting, onRequest)
            AnimatedVisibility(showHint) {
                Text(stringResource(R.string.m_teasers_swipe_hint), color = Color.White.copy(alpha = 0.7f), fontSize = Tokens.FontSize.caption, textAlign = TextAlign.Start)
            }
        }

        // Action column on the right.
        Column(
            Modifier.align(Alignment.BottomEnd).navigationBarsPadding().padding(end = Tokens.Spacing.sm, bottom = Tokens.Spacing.xl),
            verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.md),
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            Action(if (muted) Icons.Outlined.VolumeOff else Icons.Outlined.VolumeUp, stringResource(if (muted) R.string.m_teasers_unmute else R.string.m_teasers_mute), onToggleMute)
            Action(
                if (teaser.watchlisted) Icons.Filled.Bookmark else Icons.Outlined.BookmarkBorder,
                stringResource(if (teaser.watchlisted) R.string.actions_on_watchlist else R.string.actions_watchlist), onWatchlist,
            )
            Action(Icons.Outlined.Info, stringResource(R.string.media_view_details), onDetails)
            Action(Icons.Outlined.VisibilityOff, stringResource(R.string.media_not_interested), onHide)
        }
    }
}

@Composable
private fun RequestButton(teaser: Teaser, mayRequest: Boolean, requesting: Boolean, onRequest: () -> Unit) {
    val state = teaser.item.state
    when {
        state != null -> TipsarrButton(stringResource(state.title), {}, kind = ButtonKind.Secondary, compact = true, enabled = false)
        mayRequest -> TipsarrButton(stringResource(if (requesting) R.string.media_requesting else R.string.media_request), onRequest, compact = true, enabled = !requesting)
    }
}

@Composable
private fun Action(icon: androidx.compose.ui.graphics.vector.ImageVector, label: String, onClick: () -> Unit) {
    IconButton(onClick, Modifier.width(Tokens.Size.touchTarget + 8.dp).background(Color.Black.copy(alpha = 0.35f), androidx.compose.foundation.shape.CircleShape)) {
        Icon(icon, contentDescription = label, tint = Color.White)
    }
}

/** Hooks the page calls from JavaScript (they arrive on a background thread). */
private class PlayerBridge(private val onPlaying: (Boolean) -> Unit, private val onUnplayable: () -> Unit) {
    private val main = Handler(Looper.getMainLooper())

    @JavascriptInterface fun state(code: Int) { main.post { onPlaying(code == 1) } }

    /** 100, 101 and 150: removed, or the owner does not allow embedding. */
    @JavascriptInterface fun error(code: Int) { main.post { onUnplayable() } }
}

/**
 * One trailer in the YouTube IFrame player inside a WebView (YouTube allows no other way to play it).
 * Autoplays; sound follows [muted]. The WebView is paused with the app and destroyed when the card leaves.
 */
@SuppressLint("SetJavaScriptEnabled")
@Composable
private fun YouTubePlayer(key: String, muted: Boolean, modifier: Modifier, onPlaying: (Boolean) -> Unit, onUnplayable: () -> Unit) {
    val owner = LocalLifecycleOwner.current
    var web by remember { mutableStateOf<WebView?>(null) }
    AndroidView(
        modifier = modifier,
        factory = { context ->
            WebView(context).apply {
                setBackgroundColor(android.graphics.Color.BLACK)
                settings.javaScriptEnabled = true
                settings.domStorageEnabled = true
                settings.mediaPlaybackRequiresUserGesture = false
                webChromeClient = WebChromeClient()
                addJavascriptInterface(PlayerBridge(onPlaying, onUnplayable), "Android")
                loadDataWithBaseURL("https://tipsarr.local/", playerPage(key, muted), "text/html", "utf-8", null)
                web = this
            }
        },
        update = { it.evaluateJavascript("if (typeof setMuted === 'function') setMuted($muted)", null) },
        onRelease = { it.destroy() },
    )
    DisposableEffect(owner, web) {
        val observer = LifecycleEventObserver { _, event ->
            when (event) {
                Lifecycle.Event.ON_PAUSE -> { web?.evaluateJavascript("if (typeof pauseVideo === 'function') pauseVideo()", null); web?.onPause() }
                Lifecycle.Event.ON_RESUME -> { web?.onResume(); web?.evaluateJavascript("if (typeof playVideo === 'function') playVideo()", null) }
                else -> {}
            }
        }
        owner.lifecycle.addObserver(observer)
        onDispose { owner.lifecycle.removeObserver(observer) }
    }
}

private fun playerPage(key: String, muted: Boolean): String = """
<!doctype html><html><head><meta name="viewport" content="width=device-width,initial-scale=1">
<style>html,body{margin:0;height:100%;background:#000;overflow:hidden}#p{position:absolute;inset:0;width:100%;height:100%}</style></head>
<body><div id="p"></div>
<script src="https://www.youtube.com/iframe_api"></script>
<script>
var player, muted = $muted;
function onYouTubeIframeAPIReady() {
  player = new YT.Player('p', {
    videoId: '$key',
    playerVars: {autoplay: 1, controls: 0, playsinline: 1, rel: 0, modestbranding: 1, loop: 1, playlist: '$key', iv_load_policy: 3, fs: 0, disablekb: 1, origin: 'https://tipsarr.local'},
    events: {
      onReady: function (e) { if (muted) e.target.mute(); else e.target.unMute(); e.target.playVideo(); },
      onStateChange: function (e) { Android.state(e.data); },
      onError: function (e) { Android.error(e.data); }
    }
  });
}
function setMuted(m) { muted = m; if (player && player.mute) { if (m) player.mute(); else player.unMute(); } }
function pauseVideo() { if (player && player.pauseVideo) player.pauseVideo(); }
function playVideo() { if (player && player.playVideo) player.playVideo(); }
</script></body></html>
"""
