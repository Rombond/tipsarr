package com.brebond.tipsarr.features.detail

import android.content.Intent
import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.defaultMinSize
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.outlined.ArrowBack
import androidx.compose.material.icons.automirrored.outlined.KeyboardArrowRight
import androidx.compose.material.icons.outlined.Add
import androidx.compose.material.icons.outlined.Bookmark
import androidx.compose.material.icons.outlined.BookmarkBorder
import androidx.compose.material.icons.outlined.Cancel
import androidx.compose.material.icons.outlined.Flag
import androidx.compose.material.icons.outlined.MoreHoriz
import androidx.compose.material.icons.outlined.PlayArrow
import androidx.compose.material.icons.outlined.PlayCircle
import androidx.compose.material.icons.outlined.Refresh
import androidx.compose.material.icons.outlined.Replay
import androidx.compose.material.icons.outlined.Share
import androidx.compose.material.icons.outlined.Visibility
import androidx.compose.material.icons.outlined.VisibilityOff
import androidx.compose.material.icons.outlined.WarningAmber
import androidx.compose.material.icons.outlined.Person
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.heading
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontStyle
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.core.net.toUri
import com.brebond.tipsarr.R
import com.brebond.tipsarr.core.api.ApiError
import com.brebond.tipsarr.core.api.MediaDetail
import com.brebond.tipsarr.core.api.MediaItem
import com.brebond.tipsarr.core.api.MediaType
import com.brebond.tipsarr.core.api.RatingsSummary
import com.brebond.tipsarr.core.images.RemoteImage
import com.brebond.tipsarr.core.images.TmdbSize
import com.brebond.tipsarr.core.support.LocalToast
import com.brebond.tipsarr.core.support.ToastKind
import com.brebond.tipsarr.core.support.formatDate
import com.brebond.tipsarr.core.support.formatRuntime
import com.brebond.tipsarr.core.support.languageName
import com.brebond.tipsarr.core.support.plural
import com.brebond.tipsarr.core.support.stringByName
import com.brebond.tipsarr.design.Tokens
import com.brebond.tipsarr.design.palette
import com.brebond.tipsarr.ui.Banner
import com.brebond.tipsarr.ui.BannerKind
import com.brebond.tipsarr.ui.ButtonKind
import com.brebond.tipsarr.ui.MediaPoster
import com.brebond.tipsarr.ui.OfflineState
import com.brebond.tipsarr.ui.ProgressBar
import com.brebond.tipsarr.ui.RequestState
import com.brebond.tipsarr.ui.Skeleton
import com.brebond.tipsarr.ui.SkeletonBlock
import com.brebond.tipsarr.ui.StateView
import com.brebond.tipsarr.ui.StatusBadge
import com.brebond.tipsarr.ui.TipsarrButton
import kotlinx.coroutines.launch
import java.util.Locale

private val GUTTER = Tokens.Spacing.lg

private enum class DetailSheet { Request, Report }

/** Detail of a movie or show: header, ratings, request button, overview, seasons, facts, cast, recommendations. */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun MediaDetailScreen(model: MediaDetailModel, serverUrl: String, onBack: () -> Unit, onOpen: (MediaRoute) -> Unit, modifier: Modifier = Modifier) {
    val context = LocalContext.current
    val toast = LocalToast.current
    val scope = rememberCoroutineScope()
    var sheet by remember { mutableStateOf<DetailSheet?>(null) }
    var confirmCancel by remember { mutableStateOf(false) }
    var refreshing by remember { mutableStateOf(false) }
    LaunchedEffect(model) { model.load() }

    fun fail(error: Throwable) = toast?.show(ApiError.from(error).message(context), ToastKind.Error)
    fun run(work: suspend () -> Unit) { scope.launch { try { work() } catch (e: ApiError) { fail(e) } } }
    fun announce(record: com.brebond.tipsarr.core.api.RequestRecord) {
        val key = if (record.state == RequestState.Requested) "media_toast_requested" else if (record.dryRun) "media_toast_approved_dry" else "media_toast_approved"
        toast?.show((context.stringByName(key) ?: "").replace("%1\$s", model.route.title).replace("%s", model.route.title))
    }
    fun startRequest() {
        scope.launch {
            try {
                if (model.prepareRequest()) {
                    sheet = DetailSheet.Request
                } else {
                    val seasons = if (model.route.type == MediaType.Tv) model.regularSeasons.map { it.number } else null
                    announce(model.submitRequest(seasons, null, null))
                }
            } catch (e: ApiError) { fail(e) }
        }
    }
    fun share() {
        val detail = model.detail
        val web = serverUrl.trimEnd('/') + "/media/${if (model.route.type == MediaType.Movie) "movie" else "tv"}/${model.route.tmdbId}"
        val title = listOfNotNull(detail?.title ?: model.route.title, detail?.year?.let { "($it)" }).joinToString(" ")
        context.startActivity(
            Intent.createChooser(
                Intent(Intent.ACTION_SEND).setType("text/plain").putExtra(Intent.EXTRA_SUBJECT, title).putExtra(Intent.EXTRA_TEXT, "$title\n$web"),
                null,
            ),
        )
    }

    Box(modifier.fillMaxSize().background(Tokens.palette.bg)) {
        when (val phase = model.phase) {
            MediaDetailModel.Phase.Loading -> DetailSkeleton(model.route.title)
            is MediaDetailModel.Phase.Failed -> Box(Modifier.fillMaxSize().statusBarsPadding().padding(top = 56.dp)) {
                if (phase.error == ApiError.Unreachable) OfflineState({ scope.launch { model.load() } })
                else StateView(Icons.Outlined.WarningAmber, phase.error.message(context), actionTitle = stringResource(R.string.common_retry), onAction = { scope.launch { model.load() } })
            }
            MediaDetailModel.Phase.Loaded -> model.detail?.let { detail ->
                PullToRefreshBox(
                    isRefreshing = refreshing,
                    onRefresh = { scope.launch { refreshing = true; model.load(); refreshing = false } },
                    modifier = Modifier.fillMaxSize(),
                ) {
                    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState()).navigationBarsPadding(), horizontalAlignment = Alignment.CenterHorizontally) {
                        DetailHeader(detail, model.badge)
                        Column(
                            Modifier.widthIn(max = 720.dp).fillMaxWidth().padding(horizontal = GUTTER).padding(bottom = Tokens.Spacing.x3xl),
                            verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.xl),
                        ) {
                            RatingsStrip(model.ratings)
                            ActionRow(
                                model = model, detail = detail,
                                onRequest = ::startRequest,
                                onRetry = { run { model.retryRequest() } },
                                onWatchlist = { run { model.toggleWatchlist() } },
                                onHidden = { run { model.toggleHidden() } },
                                onReport = { sheet = DetailSheet.Report },
                                onCancel = { confirmCancel = true },
                            )
                            Banners(model)
                            Overview(detail)
                            if (detail.type == MediaType.Tv && model.regularSeasons.isNotEmpty()) {
                                SeasonList(model.api, model.route.tmdbId, model.regularSeasons, model.coveredSeasons, model.request?.state ?: RequestState.Requested)
                            }
                            Facts(detail)
                            if (detail.cast.isNotEmpty()) Cast(detail)
                            if (detail.recommendations.isNotEmpty()) Recommendations(detail.recommendations) { onOpen(MediaRoute(it.type, it.tmdbId, it.title)) }
                            ReportRow { sheet = DetailSheet.Report }
                        }
                    }
                }
            }
        }
        // Back and share float over the backdrop.
        Row(Modifier.fillMaxWidth().statusBarsPadding().padding(horizontal = Tokens.Spacing.sm, vertical = Tokens.Spacing.xs), horizontalArrangement = Arrangement.SpaceBetween) {
            RoundIcon(Icons.AutoMirrored.Outlined.ArrowBack, stringResource(R.string.common_back), onBack)
            if (model.phase is MediaDetailModel.Phase.Loaded) RoundIcon(Icons.Outlined.Share, stringResource(R.string.m_detail_share), ::share)
        }
    }

    when (sheet) {
        DetailSheet.Request -> RequestSheet(model, onDismiss = { sheet = null }) { record -> sheet = null; announce(record) }
        DetailSheet.Report -> ReportIssueSheet(model.route.title, onDismiss = { sheet = null }) { kind, message ->
            model.report(kind, message)
            sheet = null
            toast?.show(context.stringByName("issue_toast_reported").orEmpty())
        }
        null -> {}
    }
    if (confirmCancel) {
        AlertDialog(
            onDismissRequest = { confirmCancel = false },
            title = { Text(stringResource(R.string.m_request_cancel_confirm, model.route.title)) },
            confirmButton = { TextButton({ confirmCancel = false; run { model.cancelRequest() } }) { Text(stringResource(R.string.m_request_cancel), color = Tokens.palette.destructive) } },
            dismissButton = { TextButton({ confirmCancel = false }) { Text(stringResource(R.string.m_request_keep)) } },
            containerColor = Tokens.palette.card,
        )
    }
}

@Composable
private fun RoundIcon(icon: ImageVector, description: String, onClick: () -> Unit) {
    IconButton(onClick, Modifier.size(Tokens.Size.touchTarget).clip(CircleShape).background(Color.Black.copy(alpha = 0.45f))) {
        Icon(icon, contentDescription = description, tint = Color.White)
    }
}

/** Backdrop, poster, title, year / runtime and the state badge. */
@Composable
private fun DetailHeader(detail: MediaDetail, badge: RequestState?) {
    val context = LocalContext.current
    val meta = buildList {
        detail.year?.let { add(it) }
        if (detail.type == MediaType.Tv) {
            detail.numberOfSeasons?.takeIf { it > 0 }?.let { add(context.plural("m_detail_seasons_count", it)) }
        } else detail.runtimeMinutes?.takeIf { it > 0 }?.let { add(formatRuntime(it)) }
    }.joinToString(" · ")
    Box(Modifier.fillMaxWidth()) {
        Column {
            Box(Modifier.fillMaxWidth().height(260.dp).background(Tokens.palette.muted)) {
                RemoteImage(detail.backdropPath ?: detail.posterPath, TmdbSize.W780, Modifier.fillMaxSize())
                Box(Modifier.fillMaxSize().background(Brush.verticalGradient(0.5f to Color.Transparent, 1f to Tokens.palette.bg)))
            }
            Spacer(Modifier.height(116.dp))
        }
        Row(
            Modifier.align(Alignment.BottomStart).padding(horizontal = GUTTER),
            horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.lg),
            verticalAlignment = Alignment.Bottom,
        ) {
            Box(
                Modifier.width(120.dp).height(120.dp * Tokens.Size.posterRatio)
                    .shadow(10.dp, RoundedCornerShape(Tokens.Radius.md)).clip(RoundedCornerShape(Tokens.Radius.md))
                    .background(Tokens.palette.muted).border(BorderStroke(1.dp, Tokens.palette.border), RoundedCornerShape(Tokens.Radius.md)),
            ) { RemoteImage(detail.posterPath, TmdbSize.W342, Modifier.fillMaxSize()) }
            Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm)) {
                Text(detail.title, fontSize = Tokens.FontSize.title2, fontWeight = FontWeight.Bold, color = Tokens.palette.fg, modifier = Modifier.semantics { heading() })
                if (meta.isNotEmpty()) Text(meta, fontSize = Tokens.FontSize.footnote, color = Tokens.palette.mutedFg)
                if (badge != null) StatusBadge(badge)
            }
        }
    }
}

@Composable
private fun DetailSkeleton(title: String) {
    Skeleton(Modifier.fillMaxSize().statusBarsPadding().padding(horizontal = GUTTER)) {
        Column(Modifier.padding(top = 200.dp), verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.xl)) {
            Row(horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.lg), verticalAlignment = Alignment.Bottom) {
                Box(Modifier.width(120.dp).height(120.dp * Tokens.Size.posterRatio).clip(RoundedCornerShape(Tokens.Radius.md)).background(Tokens.palette.border))
                Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm)) {
                    Text(title, fontSize = Tokens.FontSize.title2, fontWeight = FontWeight.Bold, color = Tokens.palette.fg)
                    SkeletonBlock(Modifier.width(90.dp), height = 12.dp)
                }
            }
            SkeletonBlock(height = 48.dp, cornerRadius = Tokens.Radius.md)
            SkeletonBlock(height = 50.dp, cornerRadius = Tokens.Radius.md)
            repeat(4) { SkeletonBlock(height = 14.dp) }
        }
    }
}

/** One row of scores between the poster and the request button. Movies: TMDB, IMDb, Rotten Tomatoes, Metacritic. Shows: TMDB. */
@Composable
private fun RatingsStrip(ratings: RatingsSummary) {
    val entries = buildList {
        ratings.tmdb?.let { add("TMDB" to String.format(Locale.getDefault(), "%.1f", it)) }
        ratings.imdb?.let { add("IMDb" to String.format(Locale.getDefault(), "%.1f", it)) }
        ratings.rottenTomatoes?.let { add("RT" to "${it.toInt()}%") }
        ratings.metacritic?.let { add("MC" to it.toInt().toString()) }
    }
    if (entries.isEmpty()) return
    Row(
        Modifier.fillMaxWidth().clip(RoundedCornerShape(Tokens.Radius.md)).background(Tokens.palette.muted).padding(vertical = Tokens.Spacing.md),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        entries.forEachIndexed { index, (name, value) ->
            if (index > 0) Box(Modifier.width(1.dp).height(28.dp).background(Tokens.palette.border))
            Column(Modifier.weight(1f), horizontalAlignment = Alignment.CenterHorizontally) {
                Text(value, fontSize = Tokens.FontSize.headline, fontWeight = FontWeight.Bold, color = Tokens.palette.fg)
                Text(name, fontSize = Tokens.FontSize.caption, color = Tokens.palette.mutedFg)
            }
        }
    }
}

@Composable
private fun ActionRow(
    model: MediaDetailModel,
    detail: MediaDetail,
    onRequest: () -> Unit,
    onRetry: () -> Unit,
    onWatchlist: () -> Unit,
    onHidden: () -> Unit,
    onReport: () -> Unit,
    onCancel: () -> Unit,
) {
    val context = LocalContext.current
    var menu by remember { mutableStateOf(false) }
    val action = model.action
    Column(verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.md)) {
        Row(horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.md), verticalAlignment = Alignment.CenterVertically) {
            Box(Modifier.weight(1f)) { MainButton(action, model, onRequest, onRetry, onCancel) }
            Box {
                IconButton(
                    { menu = true },
                    Modifier.size(Tokens.Size.touchTarget).clip(CircleShape).background(Tokens.palette.muted),
                ) { Icon(Icons.Outlined.MoreHoriz, stringResource(R.string.common_more_actions), tint = Tokens.palette.fg) }
                DropdownMenu(menu, { menu = false }, containerColor = Tokens.palette.card) {
                    MenuItem(
                        if (model.flags.watchlisted) Icons.Outlined.Bookmark else Icons.Outlined.BookmarkBorder,
                        stringResource(if (model.flags.watchlisted) R.string.actions_on_watchlist else R.string.actions_watchlist),
                    ) { menu = false; onWatchlist() }
                    MenuItem(
                        if (model.flags.blocklisted) Icons.Outlined.Visibility else Icons.Outlined.VisibilityOff,
                        stringResource(if (model.flags.blocklisted) R.string.actions_show_again else R.string.media_not_interested),
                    ) { menu = false; onHidden() }
                    detail.trailerKey?.let { key ->
                        MenuItem(Icons.Outlined.PlayCircle, stringResource(R.string.actions_trailer)) {
                            menu = false
                            context.startActivity(Intent(Intent.ACTION_VIEW, "https://www.youtube.com/watch?v=$key".toUri()))
                        }
                    }
                    MenuItem(Icons.Outlined.Flag, stringResource(R.string.actions_report_short)) { menu = false; onReport() }
                    if (action is DetailAction.InProgress && action.cancellable) {
                        MenuItem(Icons.Outlined.Cancel, stringResource(R.string.m_request_cancel), Tokens.palette.destructive) { menu = false; onCancel() }
                    }
                }
            }
        }
        val request = model.request
        if (action is DetailAction.InProgress && action.state == RequestState.Downloading && request != null && request.progressPercent != null) {
            ProgressBar(request.progressPercent)
            Row(horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm)) {
                Text(stringResource(R.string.req_stage_downloading_pct, request.progressPercent.toString()), fontSize = Tokens.FontSize.footnote, color = Tokens.palette.mutedFg)
                request.etaSeconds?.takeIf { it > 0 }?.let { Text(formatEta(it), fontSize = Tokens.FontSize.footnote, color = Tokens.palette.mutedFg) }
            }
        }
    }
}

internal fun formatEta(seconds: Int): String {
    val minutes = (seconds + 30) / 60
    return if (minutes >= 60) "${minutes / 60}h ${minutes % 60}m" else "${maxOf(minutes, 1)}m"
}

@Composable
private fun MenuItem(icon: ImageVector, text: String, color: Color = Tokens.palette.fg, onClick: () -> Unit) {
    DropdownMenuItem(
        text = { Text(text, color = color) },
        leadingIcon = { Icon(icon, contentDescription = null, tint = color) },
        onClick = onClick,
    )
}

@Composable
private fun MainButton(action: DetailAction, model: MediaDetailModel, onRequest: () -> Unit, onRetry: () -> Unit, onCancel: () -> Unit) {
    val context = LocalContext.current
    when (action) {
        DetailAction.Request -> TipsarrButton(
            stringResource(if (model.busy) R.string.media_requesting else R.string.media_request), onRequest,
            fullWidth = true, enabled = !model.busy,
        ) { Icon(Icons.Outlined.Add, null, Modifier.size(20.dp)); Spacer(Modifier.width(Tokens.Spacing.sm)) }
        is DetailAction.InProgress -> TipsarrButton(
            stringResource(action.state.title), { if (action.cancellable) onCancel() },
            kind = ButtonKind.Secondary, fullWidth = true, enabled = action.cancellable,
        ) { Icon(action.state.icon, null, Modifier.size(20.dp)); Spacer(Modifier.width(Tokens.Spacing.sm)) }
        is DetailAction.Available -> if (action.watchUrl != null) {
            TipsarrButton(stringResource(R.string.m_detail_open_jellyfin), {
                context.startActivity(Intent(Intent.ACTION_VIEW, action.watchUrl.toUri()))
            }, fullWidth = true) { Icon(Icons.Outlined.PlayArrow, null, Modifier.size(20.dp)); Spacer(Modifier.width(Tokens.Spacing.sm)) }
        } else StatusLine(RequestState.Available)
        DetailAction.RequestAgain -> TipsarrButton(
            stringResource(R.string.m_detail_request_again), onRequest, kind = ButtonKind.Secondary, fullWidth = true, enabled = !model.busy,
        ) { Icon(Icons.Outlined.Replay, null, Modifier.size(20.dp)); Spacer(Modifier.width(Tokens.Spacing.sm)) }
        is DetailAction.Failed -> if (action.canRetry) {
            TipsarrButton(stringResource(R.string.common_retry), onRetry, fullWidth = true, enabled = !model.busy) {
                Icon(Icons.Outlined.Refresh, null, Modifier.size(20.dp)); Spacer(Modifier.width(Tokens.Spacing.sm))
            }
        } else StatusLine(RequestState.Failed)
    }
}

@Composable
private fun StatusLine(state: RequestState) {
    Row(
        Modifier.fillMaxWidth().defaultMinSize(minHeight = Tokens.Size.touchTarget),
        horizontalArrangement = Arrangement.Center, verticalAlignment = Alignment.CenterVertically,
    ) {
        Icon(state.icon, null, tint = state.color, modifier = Modifier.size(20.dp))
        Spacer(Modifier.width(Tokens.Spacing.sm))
        Text(stringResource(state.title), fontSize = Tokens.FontSize.headline, fontWeight = FontWeight.SemiBold, color = state.color)
    }
}

@Composable
private fun Banners(model: MediaDetailModel) {
    when (model.action) {
        DetailAction.RequestAgain -> {
            val reason = model.request?.declineReason
            Banner(if (reason != null) stringResource(R.string.req_reason, reason) else stringResource(R.string.m_detail_declined), kind = BannerKind.Error)
        }
        is DetailAction.Failed -> Banner(stringResource(R.string.req_failed_generic), kind = BannerKind.Warning)
        else -> {}
    }
}

@Composable
private fun SectionTitle(text: String) {
    Text(text, fontSize = Tokens.FontSize.title3, fontWeight = FontWeight.Bold, color = Tokens.palette.fg, modifier = Modifier.semantics { heading() })
}

@OptIn(ExperimentalLayoutApi::class)
@Composable
private fun Overview(detail: MediaDetail) {
    Column(verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.md)) {
        SectionTitle(stringResource(R.string.detail_overview))
        detail.tagline?.takeIf { it.isNotEmpty() }?.let { Text(it, fontSize = Tokens.FontSize.subhead, fontStyle = FontStyle.Italic, color = Tokens.palette.mutedFg) }
        val overview = detail.overview?.takeIf { it.isNotEmpty() }
        if (overview != null) Text(overview, fontSize = Tokens.FontSize.callout, lineHeight = Tokens.FontSize.callout * 1.4f, color = Tokens.palette.fg)
        else Text(stringResource(R.string.m_detail_no_overview), fontSize = Tokens.FontSize.callout, color = Tokens.palette.mutedFg)
        if (detail.genres.isNotEmpty()) {
            FlowRow(horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm), verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm)) {
                detail.genres.forEach { genre ->
                    Text(
                        genre.name, fontSize = Tokens.FontSize.footnote, fontWeight = FontWeight.Medium, color = Tokens.palette.fg,
                        modifier = Modifier.clip(CircleShape).background(Tokens.palette.muted).padding(horizontal = Tokens.Spacing.md, vertical = Tokens.Spacing.xs + 2.dp),
                    )
                }
            }
        }
    }
}

@Composable
private fun Facts(detail: MediaDetail) {
    val context = LocalContext.current
    Column(verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.md)) {
        SectionTitle(stringResource(R.string.detail_title_fallback))
        Column {
            detail.status?.takeIf { it.isNotEmpty() }?.let { FactRow(stringResource(R.string.fact_status), context.stringByName("status_$it") ?: it) }
            detail.releaseDate?.takeIf { it.isNotEmpty() }?.let { FactRow(stringResource(R.string.fact_release_date), formatDate(it)) }
            detail.runtimeMinutes?.takeIf { it > 0 }?.let { FactRow(stringResource(R.string.fact_runtime), formatRuntime(it)) }
            detail.originalLanguage?.let { code -> languageName(code)?.let { FactRow(stringResource(R.string.fact_language), it) } }
            detail.studios?.takeIf { it.isNotEmpty() }?.let {
                FactRow(stringResource(if (detail.type == MediaType.Tv) R.string.fact_network else R.string.fact_studio), it.take(3).joinToString(", "))
            }
        }
    }
}

@Composable
private fun FactRow(label: String, value: String) {
    Column {
        Row(Modifier.fillMaxWidth().padding(vertical = Tokens.Spacing.sm), verticalAlignment = Alignment.Top) {
            Text(label, fontSize = Tokens.FontSize.subhead, color = Tokens.palette.mutedFg)
            Spacer(Modifier.width(Tokens.Spacing.lg))
            Text(value, fontSize = Tokens.FontSize.subhead, color = Tokens.palette.fg, textAlign = TextAlign.End, modifier = Modifier.weight(1f))
        }
        HorizontalDivider(color = Tokens.palette.border)
    }
}

@Composable
private fun Cast(detail: MediaDetail) {
    Column(verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.md)) {
        SectionTitle(stringResource(R.string.detail_cast))
        LazyRow(horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.md)) {
            items(detail.cast.take(15), key = { it.id }) { member ->
                Column(Modifier.width(80.dp), horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.xs)) {
                    Box(Modifier.size(64.dp).clip(CircleShape).background(Tokens.palette.muted), contentAlignment = Alignment.Center) {
                        Icon(Icons.Outlined.Person, null, tint = Tokens.palette.mutedFg)
                        RemoteImage(member.profilePath, TmdbSize.W185, Modifier.fillMaxSize())
                    }
                    Text(member.name, fontSize = Tokens.FontSize.caption, fontWeight = FontWeight.Medium, color = Tokens.palette.fg, maxLines = 1, overflow = TextOverflow.Ellipsis)
                    member.character?.takeIf { it.isNotEmpty() }?.let {
                        Text(it, fontSize = 11.sp, color = Tokens.palette.mutedFg, maxLines = 1, overflow = TextOverflow.Ellipsis)
                    }
                }
            }
        }
    }
}

@Composable
private fun Recommendations(items: List<MediaItem>, onOpen: (MediaItem) -> Unit) {
    Column(verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.md)) {
        SectionTitle(stringResource(R.string.detail_more_like))
        LazyRow(horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.md)) {
            items(items.take(15), key = { it.id }) { item -> MediaPoster(item, { onOpen(item) }, Modifier.width(110.dp)) }
        }
    }
}

@Composable
private fun ReportRow(onClick: () -> Unit) {
    Row(
        Modifier.fillMaxWidth().defaultMinSize(minHeight = Tokens.Size.touchTarget + Tokens.Spacing.sm)
            .clip(RoundedCornerShape(Tokens.Radius.md)).background(Tokens.palette.card)
            .border(BorderStroke(1.dp, Tokens.palette.border), RoundedCornerShape(Tokens.Radius.md))
            .clickable(onClick = onClick).padding(horizontal = Tokens.Spacing.lg),
        horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.md), verticalAlignment = Alignment.CenterVertically,
    ) {
        Icon(Icons.Outlined.Flag, null, tint = Tokens.palette.fg)
        Text(stringResource(R.string.actions_report_short), fontSize = Tokens.FontSize.body, fontWeight = FontWeight.Medium, color = Tokens.palette.fg, modifier = Modifier.weight(1f))
        Icon(Icons.AutoMirrored.Outlined.KeyboardArrowRight, null, tint = Tokens.palette.mutedFg)
    }
}
