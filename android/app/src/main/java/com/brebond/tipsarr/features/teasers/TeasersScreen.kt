package com.brebond.tipsarr.features.teasers

import android.content.Context
import android.content.ContextWrapper
import android.media.AudioManager
import androidx.activity.ComponentActivity
import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.animateContentSize
import androidx.compose.animation.core.animateDpAsState
import androidx.compose.animation.core.animateFloatAsState
import androidx.compose.animation.core.snap
import androidx.compose.animation.core.spring
import androidx.compose.foundation.clickable
import androidx.compose.foundation.gestures.awaitEachGesture
import androidx.compose.foundation.gestures.awaitFirstDown
import androidx.compose.foundation.gestures.waitForUpOrCancellation
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.asPaddingValues
import androidx.compose.foundation.layout.calculateStartPadding
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.safeDrawing
import androidx.compose.foundation.layout.statusBars
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.FastForward
import androidx.compose.material.icons.outlined.Visibility
import androidx.compose.runtime.mutableFloatStateOf
import androidx.compose.ui.graphics.TransformOrigin
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.input.pointer.PointerEventPass
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.platform.LocalConfiguration
import androidx.compose.ui.platform.LocalLayoutDirection
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.geometry.Offset
import com.brebond.tipsarr.ui.ScoreLabel
import kotlinx.coroutines.delay
import kotlinx.coroutines.withTimeoutOrNull
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
import androidx.compose.foundation.layout.offset
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
import androidx.compose.runtime.rememberUpdatedState
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.runtime.snapshotFlow
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.draw.blur
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
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
import com.brebond.tipsarr.core.support.TableTop
import com.brebond.tipsarr.core.support.foldOffset
import com.brebond.tipsarr.core.support.rememberTableTop
import androidx.compose.foundation.layout.ColumnScope
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
    onDisabled: () -> Unit = {},
) {
    val scope = rememberCoroutineScope()
    LaunchedEffect(model) { model.loadIfNeeded() }
    // While this screen is up it owns the players: the background preparation stands aside.
    DisposableEffect(model) {
        model.screenShown = true
        onDispose { model.screenShown = false }
    }
    // The server turned trailer files off: the tab goes away (the caller shows a message and returns to Discover).
    // Trailers still being prepared: ask again every 8 s. Ends with the composition (tab left) or once cards arrive.
    LaunchedEffect(model, model.preparing) { model.pollWhilePreparing() }
    LaunchedEffect(model.disabled) { if (model.disabled) onDisabled() }
    // Sound follows the media volume while this screen is shown; a pinch-hidden interface does not outlive it.
    val context = LocalContext.current
    DisposableEffect(context) {
        val activity = context.findActivity()
        val previous = activity?.volumeControlStream ?: AudioManager.USE_DEFAULT_STREAM_TYPE
        activity?.volumeControlStream = AudioManager.STREAM_MUSIC
        onDispose {
            activity?.volumeControlStream = previous
            model.uiHidden = false
            model.pinching = false
        }
    }
    val tablet = LocalConfiguration.current.smallestScreenWidthDp >= 600
    val tableTop = rememberTableTop()
    var top by remember { mutableStateOf(0.dp) }
    // The whole width on a phone (also sideways); a narrow 9:16 column in the middle on a tablet.
    BoxWithConstraints(modifier.fillMaxSize().background(Color.Black).foldOffset { top = it }, contentAlignment = Alignment.Center) {
        // Half folded like a laptop: the whole width, the trailer above the fold and its controls below it.
        val fold = tableTop?.let { TableTop(it.top - top, it.bottom - top) }?.takeIf { it.top > 0.dp && it.bottom < maxHeight }
        val column = if (tablet && fold == null) minOf(maxWidth, maxHeight * 9f / 16f) else maxWidth
        val landscape = !tablet && fold == null && maxWidth > maxHeight
        Box(Modifier.width(column).fillMaxHeight()) {
            when (model.phase) {
                Phase.Idle, Phase.Loading -> Waiting(model, api, onOpenItem, preparing = false)
                is Phase.Failed -> DarkState(stringResource(R.string.m_teasers_error), stringResource(R.string.common_retry)) { scope.launch { model.refresh() } }
                Phase.Loaded -> if (model.preparing) Waiting(model, api, onOpenItem, preparing = true)
                else Feed(model, api, me, settings, landscape, fold, onOpenItem)
            }
        }
    }
}

/**
 * The first batch is not here yet (a first visit can take a while on the server): the trending titles stand in, swipe
 * sideways, tap for the title page. A spinner (and the "preparing" text once the server said so) stays on top.
 */
@Composable
private fun Waiting(model: TeasersModel, api: TipsarrApi, onOpenItem: (MediaItem) -> Unit, preparing: Boolean) {
    LaunchedEffect(model) { model.loadFallback() }
    val items = model.fallback
    if (items.isEmpty()) {
        Column(
            Modifier.fillMaxSize().padding(Tokens.Spacing.x3xl),
            verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.lg, Alignment.CenterVertically),
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            CircularProgressIndicator(color = Color.White)
            if (preparing) Text(stringResource(R.string.m_teasers_preparing), color = Color.White, fontSize = Tokens.FontSize.headline, textAlign = TextAlign.Center)
        }
        return
    }
    val pager = androidx.compose.foundation.pager.rememberPagerState { items.size }
    Box(Modifier.fillMaxSize()) {
        androidx.compose.foundation.pager.HorizontalPager(pager, Modifier.fillMaxSize()) { index ->
            val item = items[index]
            Box(Modifier.fillMaxSize().background(Color.Black).clickable { onOpenItem(item) }) {
                RemoteImage(item.backdropPath ?: item.posterPath, TmdbSize.W780, Modifier.fillMaxSize().alpha(0.7f))
                Box(Modifier.fillMaxWidth().fillMaxHeight(0.5f).align(Alignment.BottomCenter).background(Brush.verticalGradient(listOf(Color.Transparent, Color.Black.copy(alpha = 0.8f)))))
                Column(
                    Modifier.align(Alignment.BottomStart).navigationBarsPadding().padding(Tokens.Spacing.lg).padding(bottom = Tokens.Spacing.xl),
                    verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm),
                ) {
                    Text(item.title, color = Color.White, fontSize = Tokens.FontSize.title2, fontWeight = FontWeight.Bold, maxLines = 2, overflow = TextOverflow.Ellipsis)
                    item.releaseYear?.let { Text(it, color = Color.White.copy(alpha = 0.85f), fontSize = Tokens.FontSize.subhead) }
                    item.overview?.takeIf { it.isNotBlank() }?.let { Text(it, color = Color.White.copy(alpha = 0.85f), fontSize = Tokens.FontSize.subhead, maxLines = 3, overflow = TextOverflow.Ellipsis) }
                }
            }
        }
        Row(
            Modifier.align(Alignment.TopCenter).statusBarsPadding().padding(top = Tokens.Spacing.lg).padding(horizontal = Tokens.Spacing.lg)
                .background(Color.Black.copy(alpha = 0.55f), CircleShape).padding(horizontal = Tokens.Spacing.md, vertical = Tokens.Spacing.sm),
            horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            CircularProgressIndicator(Modifier.size(16.dp), color = Color.White, strokeWidth = 2.dp)
            Text(stringResource(R.string.m_teasers_preparing), color = Color.White, fontSize = Tokens.FontSize.footnote)
        }
    }
}

private tailrec fun Context.findActivity(): ComponentActivity? = when (this) {
    is ComponentActivity -> this
    is ContextWrapper -> baseContext.findActivity()
    else -> null
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
private fun Feed(model: TeasersModel, api: TipsarrApi, me: () -> Profile, settings: AppSettings, landscape: Boolean, fold: TableTop?, onOpenItem: (MediaItem) -> Unit) {
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
            model.uiHidden = false // the interface comes back with the next trailer
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

    VerticalPager(pager, Modifier.fillMaxSize(), beyondViewportPageCount = 1, userScrollEnabled = !model.pinching, key = { index -> model.items.getOrNull(index)?.id ?: "end" }) { index ->
        val teaser = model.items.getOrNull(index)
        if (teaser == null) {
            DarkState(stringResource(R.string.m_teasers_done))
        } else {
            TeaserCard(
                teaser = teaser,
                active = index == settled && resumed,
                preload = index == settled + 1,
                api = api,
                takePrepared = model::takePrepared,
                muted = settings.teaserMuted,
                showHint = index == 0 && !hintGone,
                landscape = landscape,
                fold = fold,
                uiHidden = model.uiHidden,
                onUiHidden = { model.uiHidden = it },
                onPinching = { model.pinching = it },
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

/** Hold on a side of the page for this long: 2x speed. */
private const val HOLD_MS = 250L
/** Share of the width, each side, where a hold speeds the trailer up. */
private const val SIDE = 0.28f

@Composable
private fun TeaserCard(
    teaser: Teaser,
    active: Boolean,
    preload: Boolean,
    api: TipsarrApi,
    takePrepared: (Teaser) -> TeaserVideo?,
    muted: Boolean,
    showHint: Boolean,
    landscape: Boolean,
    fold: TableTop?,
    uiHidden: Boolean,
    mayRequest: Boolean,
    requesting: Boolean,
    onUiHidden: (Boolean) -> Unit,
    onPinching: (Boolean) -> Unit,
    onToggleMute: () -> Unit,
    onRequest: () -> Unit,
    onDetails: () -> Unit,
    onHide: () -> Unit,
    onWatchlist: () -> Unit,
    onUnplayable: () -> Unit,
) {
    // The current card and the next one hold a player (the next one buffers, paused).
    val video = if (active || preload) rememberTeaserVideo(api.serverUrl.trimEnd('/') + teaser.videoUrl, api.token, { takePrepared(teaser) }, onUnplayable) else null
    var paused by remember(teaser.id) { mutableStateOf(false) }
    var fast by remember { mutableStateOf(false) }
    var overviewOpen by remember(teaser.id) { mutableStateOf(false) }
    var pinching by remember { mutableStateOf(false) }
    var pinchScale by remember { mutableFloatStateOf(1f) }
    var pinchOrigin by remember { mutableStateOf(TransformOrigin.Center) }
    val setUiHidden by rememberUpdatedState(onUiHidden)
    val setPinching by rememberUpdatedState(onPinching)
    val hidden = uiHidden || pinching
    val scale by animateFloatAsState(pinchScale, if (pinching) snap() else spring(), label = "pinch")
    // Sideways, the notch side: the title and description keep clear of it; the action buttons do not.
    val leading = if (landscape) WindowInsets.safeDrawing.asPaddingValues().calculateStartPadding(LocalLayoutDirection.current) else 0.dp
    val bottom = Tokens.Spacing.xl + 16.dp

    LaunchedEffect(active) { if (!active) { paused = false; fast = false } }
    // The screen does not go to sleep while this trailer plays (not paused, not in the background); cleared on leave.
    val playingNow = video != null && active && !paused
    if (playingNow) {
        val view = androidx.compose.ui.platform.LocalView.current
        DisposableEffect(view) {
            view.keepScreenOn = true
            onDispose { view.keepScreenOn = false }
        }
    }
    if (video != null) {
        LaunchedEffect(video, active, paused) { video.setPlaying(active && !paused) }
        LaunchedEffect(video, muted) { video.setMuted(muted) }
        LaunchedEffect(video, fast) { video.setFast(fast) }
        LaunchedEffect(video, active) {
            while (active) {
                video.tick()
                delay(200)
            }
        }
    }

    Box(
        Modifier.fillMaxSize().background(Color.Black)
            // Two fingers: pinch the picture from 1x to 4x around the fingers. Watched before the children (Initial pass) and
            // only consuming once a second finger is down, so a single finger still taps, holds, scrubs and scrolls.
            .pointerInput(active) {
                if (!active) return@pointerInput
                awaitEachGesture {
                    awaitFirstDown(requireUnconsumed = false, pass = PointerEventPass.Initial)
                    var started = false
                    var startSpan = 1f
                    try {
                        do {
                            val event = awaitPointerEvent(PointerEventPass.Initial)
                            val down = event.changes.filter { it.pressed }
                            if (down.size >= 2) {
                                val a = down[0].position
                                val b = down[1].position
                                val span = (a - b).getDistance()
                                if (!started) {
                                    started = true
                                    startSpan = span.coerceAtLeast(1f)
                                    val mid = Offset((a.x + b.x) / 2f, (a.y + b.y) / 2f)
                                    pinchOrigin = TransformOrigin(mid.x / size.width, mid.y / size.height)
                                    fast = false
                                    pinching = true
                                    setPinching(true)
                                    setUiHidden(true)
                                }
                                pinchScale = (span / startSpan).coerceIn(1f, 4f)
                            }
                            if (started) event.changes.forEach { it.consume() }
                        } while (event.changes.any { it.pressed })
                    } finally {
                        if (started) {
                            pinching = false
                            pinchScale = 1f
                            setPinching(false)
                        }
                    }
                }
            },
    ) {
        // The picture, like iOS: the whole screen, the video fitted and centred over a blurred, dimmed backdrop.
        // Table-top: the picture fills the upper half only.
        val pictureArea = if (fold != null) Modifier.fillMaxWidth().height(fold.top) else Modifier.fillMaxSize()
        Box(pictureArea.graphicsLayer { scaleX = scale; scaleY = scale; transformOrigin = pinchOrigin }) {
            RemoteImage(teaser.backdropPath ?: teaser.posterPath, TmdbSize.W780, Modifier.fillMaxSize().alpha(0.55f).blur(24.dp))
            if (video != null) {
                TeaserSurface(video, Modifier.fillMaxSize().alpha(if (video.ready) 1f else 0f))
            }
        }
        Box(pictureArea) {
            if (active && video != null && !video.ready && !paused) {
                CircularProgressIndicator(modifier = Modifier.align(Alignment.Center), color = Color.White)
            }

            // Tap anywhere pauses or resumes; holding a side plays at 2x until the finger lifts (or the pager takes the gesture).
            Box(
                Modifier.fillMaxSize().pointerInput(active) {
                    if (!active) return@pointerInput
                    val edge = size.width * SIDE
                    awaitEachGesture {
                        val down = awaitFirstDown(requireUnconsumed = false)
                        val side = down.position.x < edge || down.position.x > size.width - edge
                        var finished = false
                        val up = withTimeoutOrNull(HOLD_MS) { waitForUpOrCancellation().also { finished = true } }
                        if (finished) {
                            if (up != null) paused = !paused
                        } else if (side) {
                            fast = true
                            try { waitForUpOrCancellation() } finally { fast = false }
                        } else if (waitForUpOrCancellation() != null) {
                            paused = !paused
                        }
                    }
                },
            )
            if (paused && active) {
                Icon(Icons.Outlined.PlayCircle, contentDescription = null, tint = Color.White.copy(alpha = 0.85f), modifier = Modifier.align(Alignment.Center).size(56.dp))
            }
        }

        if (!hidden) {
            val info: @Composable ColumnScope.() -> Unit = {
                Text(
                    teaser.title, color = Color.White, fontSize = Tokens.FontSize.title2, fontWeight = FontWeight.Bold, maxLines = 2, overflow = TextOverflow.Ellipsis,
                    modifier = Modifier.plainClickable(onDetails),
                )
                val light = Color.White.copy(alpha = 0.85f)
                Row(horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm), verticalAlignment = Alignment.CenterVertically) {
                    Text(stringResource(if (teaser.type == MediaType.Tv) R.string.m_poster_tv else R.string.m_poster_movie), color = light, fontSize = Tokens.FontSize.subhead, fontWeight = FontWeight.Medium)
                    teaser.releaseDate?.take(4)?.takeIf { it.isNotEmpty() }?.let { Text("· $it", color = light, fontSize = Tokens.FontSize.subhead, fontWeight = FontWeight.Medium) }
                    if (teaser.voteAverage > 0) {
                        Text("·", color = light, fontSize = Tokens.FontSize.subhead, fontWeight = FontWeight.Medium)
                        ScoreLabel(teaser.type, teaser.tmdbId, teaser.voteAverage, textColor = light)
                    }
                }
                if (video != null && video.scrubbing) {
                    Text(
                        "${clock(video.positionMs)} / ${clock(video.durationMs)}",
                        color = Color.White, fontSize = Tokens.FontSize.title3, fontWeight = FontWeight.SemiBold, style = TextStyle(fontFeatureSettings = "tnum"),
                    )
                } else {
                    teaser.overview?.takeIf { it.isNotBlank() }?.let { Overview(it, overviewOpen) { overviewOpen = !overviewOpen } }
                }
                // The hint shows once the first trailer played to its end three times in a row (two full loops after the first play).
                AnimatedVisibility(showHint && (video?.loops ?: 0) >= 2) {
                    Text(stringResource(R.string.m_teasers_swipe_hint), color = Color.White.copy(alpha = 0.7f), fontSize = Tokens.FontSize.footnote, textAlign = TextAlign.Start, modifier = Modifier.padding(top = Tokens.Spacing.xs))
                }
            }
            // Same order as iOS: request (or its state), details, watchlist, not interested, mute.
            val actions: @Composable () -> Unit = {
                val state = teaser.item.state
                if (state != null) {
                    Action(state.icon, stringResource(state.title), {}, tint = state.color, enabled = false)
                } else if (mayRequest) {
                    Action(Icons.Filled.Add, stringResource(R.string.media_request), onRequest, enabled = !requesting)
                }
                Action(Icons.Outlined.Info, stringResource(R.string.media_view_details), onDetails)
                Action(
                    if (teaser.watchlisted) Icons.Filled.Bookmark else Icons.Outlined.BookmarkBorder,
                    stringResource(if (teaser.watchlisted) R.string.actions_on_watchlist else R.string.actions_watchlist), onWatchlist,
                )
                Action(Icons.Outlined.VisibilityOff, stringResource(R.string.media_not_interested), onHide)
                Action(if (muted) Icons.Outlined.VolumeOff else Icons.Outlined.VolumeUp, stringResource(if (muted) R.string.m_teasers_unmute else R.string.m_teasers_mute), onToggleMute)
            }
            if (fold != null) {
                // Table-top: the progress line under the picture, then the title block and a row of actions in the lower half.
                Column(Modifier.fillMaxSize().padding(top = fold.bottom)) {
                    if (video != null && video.ready) ScrubBar(video, Modifier)
                    Column(
                        Modifier.fillMaxWidth().weight(1f).padding(start = Tokens.Spacing.lg, end = Tokens.Spacing.lg, top = Tokens.Spacing.sm, bottom = Tokens.Spacing.lg),
                        verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.lg),
                    ) {
                        Column(verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm), content = info)
                        Row(horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.lg), verticalAlignment = Alignment.CenterVertically) { actions() }
                    }
                }
            } else {
                Box(Modifier.fillMaxWidth().fillMaxHeight(0.5f).align(Alignment.BottomCenter).background(Brush.verticalGradient(listOf(Color.Transparent, Color.Black.copy(alpha = 0.75f)))))

                // Like iOS: the title block on the left, the round actions on the right, both resting on the same bottom line.
                Row(
                    Modifier.align(Alignment.BottomCenter).navigationBarsPadding().fillMaxWidth()
                        .padding(start = Tokens.Spacing.lg + leading, end = Tokens.Spacing.lg, bottom = bottom),
                    horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.lg),
                    verticalAlignment = Alignment.Bottom,
                ) {
                    Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm), content = info)
                    Column(verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.lg), horizontalAlignment = Alignment.CenterHorizontally) { actions() }
                }
            }
        }

        if (video != null && video.ready && !hidden && fold == null) ScrubBar(video, Modifier.align(Alignment.BottomCenter).navigationBarsPadding())

        if (fast) {
            Row(
                Modifier.align(Alignment.TopCenter).statusBarsPadding().padding(top = Tokens.Spacing.lg)
                    .background(Color.Black.copy(alpha = 0.5f), CircleShape).padding(horizontal = Tokens.Spacing.md, vertical = Tokens.Spacing.sm),
                horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.xs),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Icon(Icons.Filled.FastForward, contentDescription = null, tint = Color.White, modifier = Modifier.size(18.dp))
                Text("2\u00d7", color = Color.White, fontSize = Tokens.FontSize.subhead, fontWeight = FontWeight.Bold)
            }
        }

        // A pinch hid the interface: this brings it back.
        if (uiHidden && !pinching) {
            IconButton(
                { setUiHidden(false) },
                Modifier.align(Alignment.TopEnd).statusBarsPadding().padding(top = Tokens.Spacing.md, end = Tokens.Spacing.lg).size(44.dp).background(Color.Black.copy(alpha = 0.45f), CircleShape),
            ) {
                Icon(Icons.Outlined.Visibility, contentDescription = stringResource(R.string.m_teasers_show_interface), tint = Color.White)
            }
        }
    }
}

private fun clock(ms: Long): String {
    val total = (ms.coerceAtLeast(0L) / 1000L).toInt()
    return "%d:%02d".format(total / 60, total % 60)
}

/** Two lines; a tap shows all of it (scrollable when long, 180 dp at most) and a second tap folds it back. */
@Composable
private fun Overview(text: String, open: Boolean, onToggle: () -> Unit) {
    Box(Modifier.fillMaxWidth().animateContentSize().plainClickable(onToggle)) {
        if (open) {
            Box(Modifier.heightIn(max = 180.dp).verticalScroll(rememberScrollState())) {
                Text(text, color = Color.White.copy(alpha = 0.85f), fontSize = Tokens.FontSize.subhead)
            }
        } else {
            Text(text, color = Color.White.copy(alpha = 0.85f), fontSize = Tokens.FontSize.subhead, maxLines = 2, overflow = TextOverflow.Ellipsis)
        }
    }
}

/** The progress line along the bottom edge: drag it to go forward or back. It keeps the touch, so the pager does not scroll. */
@Composable
private fun ScrubBar(video: TeaserVideo, modifier: Modifier) {
    val line by animateDpAsState(if (video.scrubbing) 8.dp else 3.dp, label = "scrub")
    Box(
        modifier.fillMaxWidth().height(28.dp).pointerInput(video) {
            awaitEachGesture {
                val down = awaitFirstDown(requireUnconsumed = false)
                down.consume()
                video.beginScrub()
                video.scrub(down.position.x / size.width.coerceAtLeast(1))
                try {
                    do {
                        val event = awaitPointerEvent()
                        val change = event.changes.firstOrNull { it.id == down.id }
                        if (change != null && change.pressed) video.scrub(change.position.x / size.width.coerceAtLeast(1))
                        event.changes.forEach { it.consume() }
                    } while (event.changes.any { it.pressed })
                } finally {
                    video.endScrub()
                }
            }
        },
        contentAlignment = Alignment.BottomStart,
    ) {
        Box(Modifier.fillMaxWidth().height(line).background(Color.White.copy(alpha = 0.25f)))
        Box(Modifier.fillMaxWidth(video.progress.coerceIn(0f, 1f)).height(line).background(Color.White.copy(alpha = 0.9f)))
    }
}

/** Clickable without the ripple (text over a video). */
@Composable
private fun Modifier.plainClickable(onClick: () -> Unit): Modifier =
    clickable(interactionSource = remember { MutableInteractionSource() }, indication = null, onClick = onClick)

@Composable
private fun Action(icon: androidx.compose.ui.graphics.vector.ImageVector, label: String, onClick: () -> Unit, tint: Color = Color.White, enabled: Boolean = true) {
    IconButton(onClick, Modifier.size(48.dp).background(Color.Black.copy(alpha = 0.4f), CircleShape), enabled = enabled) {
        Icon(icon, contentDescription = label, tint = tint)
    }
}
