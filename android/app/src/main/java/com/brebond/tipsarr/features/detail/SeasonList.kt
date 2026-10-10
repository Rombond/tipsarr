package com.brebond.tipsarr.features.detail

import androidx.compose.animation.animateContentSize
import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.KeyboardArrowDown
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateMapOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.rotate
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.semantics.heading
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.brebond.tipsarr.R
import com.brebond.tipsarr.core.api.ApiError
import com.brebond.tipsarr.core.api.Episode
import com.brebond.tipsarr.core.api.SeasonInfo
import com.brebond.tipsarr.core.api.TipsarrApi
import com.brebond.tipsarr.core.api.episodes
import com.brebond.tipsarr.core.images.RemoteImage
import com.brebond.tipsarr.core.images.TmdbSize
import com.brebond.tipsarr.core.support.formatDate
import com.brebond.tipsarr.core.support.plural
import com.brebond.tipsarr.design.Tokens
import com.brebond.tipsarr.design.palette
import com.brebond.tipsarr.ui.RequestState
import com.brebond.tipsarr.ui.StatusBadge
import kotlinx.coroutines.launch
import java.util.Locale

/** Seasons of a show; a tap opens the episodes of that season (loaded on first open), as on the web. */
@Composable
fun SeasonList(api: TipsarrApi, tmdbId: Int, seasons: List<SeasonInfo>, covered: Set<Int>, state: RequestState) {
    val context = LocalContext.current
    var open by remember { mutableStateOf<Int?>(null) }
    val episodes = remember { mutableStateMapOf<Int, List<Episode>>() }
    val failed = remember { mutableStateMapOf<Int, String>() }
    var loading by remember { mutableStateOf<Int?>(null) }
    val scope = rememberCoroutineScope()

    fun toggle(number: Int) {
        if (open == number) { open = null; return }
        open = number
        if (episodes[number] != null) return
        loading = number
        failed.remove(number)
        scope.launch {
            try { episodes[number] = api.episodes(tmdbId, number) } catch (e: ApiError) { failed[number] = e.message(context) }
            loading = null
        }
    }

    Column(verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.md)) {
        Text(
            stringResource(R.string.seasons_title), fontSize = Tokens.FontSize.title3, fontWeight = FontWeight.Bold,
            color = Tokens.palette.fg, modifier = Modifier.semantics { heading() },
        )
        seasons.forEach { season ->
            val shape = RoundedCornerShape(Tokens.Radius.md)
            Column(Modifier.fillMaxWidth().clip(shape).background(Tokens.palette.card).border(BorderStroke(1.dp, Tokens.palette.border), shape).animateContentSize()) {
                Row(
                    Modifier.fillMaxWidth().clickable(role = Role.Button) { toggle(season.number) }.padding(Tokens.Spacing.md),
                    horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.md), verticalAlignment = Alignment.CenterVertically,
                ) {
                    Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
                        Text(season.name, fontSize = Tokens.FontSize.body, fontWeight = FontWeight.SemiBold, color = Tokens.palette.fg)
                        val count = context.plural("seasons_episodes", season.episodeCount)
                        val year = season.airDate?.takeIf { it.length >= 4 }?.take(4)
                        Text(if (year != null) "$count · $year" else count, fontSize = Tokens.FontSize.footnote, color = Tokens.palette.mutedFg)
                    }
                    if (season.number in covered) StatusBadge(state)
                    Icon(
                        Icons.Outlined.KeyboardArrowDown, null, tint = Tokens.palette.mutedFg,
                        modifier = Modifier.rotate(if (open == season.number) 180f else 0f),
                    )
                }
                if (open == season.number) {
                    Column(
                        Modifier.fillMaxWidth().background(Tokens.palette.muted.copy(alpha = 0.4f)).padding(Tokens.Spacing.md),
                        verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.md),
                    ) {
                        val message = failed[season.number]
                        when {
                            loading == season.number -> Text(stringResource(R.string.seasons_loading), fontSize = Tokens.FontSize.footnote, color = Tokens.palette.mutedFg)
                            message != null -> Text(message, fontSize = Tokens.FontSize.footnote, color = Tokens.palette.mutedFg)
                            else -> episodes[season.number].orEmpty().forEach { EpisodeRow(it) }
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun EpisodeRow(episode: Episode) {
    val context = LocalContext.current
    val info = listOfNotNull(
        episode.airDate?.takeIf { it.isNotEmpty() }?.let { formatDate(it, java.time.format.FormatStyle.MEDIUM) },
        episode.runtimeMinutes?.takeIf { it > 0 }?.let { context.plural("time_min", it) },
        episode.voteAverage.takeIf { it > 0 }?.let { "★ " + String.format(Locale.getDefault(), "%.1f", it) },
    ).joinToString(" · ")
    Row(horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.md), verticalAlignment = Alignment.Top) {
        Box(Modifier.width(128.dp).height(72.dp).clip(RoundedCornerShape(Tokens.Radius.sm)).background(Tokens.palette.muted), contentAlignment = Alignment.Center) {
            Text(stringResource(R.string.seasons_no_preview), fontSize = 11.sp, color = Tokens.palette.mutedFg)
            RemoteImage(episode.stillPath, TmdbSize.W342, Modifier.fillMaxSize())
            Text(
                "E${episode.number}", fontSize = 11.sp, fontWeight = FontWeight.SemiBold, color = Color.White,
                modifier = Modifier.align(Alignment.BottomStart).padding(4.dp).clip(RoundedCornerShape(4.dp)).background(Color.Black.copy(alpha = 0.7f)).padding(horizontal = 5.dp, vertical = 2.dp),
            )
        }
        Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
            Text(episode.name, fontSize = Tokens.FontSize.subhead, fontWeight = FontWeight.SemiBold, color = Tokens.palette.fg)
            if (info.isNotEmpty()) Text(info, fontSize = Tokens.FontSize.caption, color = Tokens.palette.mutedFg)
            episode.overview?.takeIf { it.isNotEmpty() }?.let {
                Text(it, fontSize = Tokens.FontSize.caption, color = Tokens.palette.mutedFg, maxLines = 3, overflow = TextOverflow.Ellipsis, modifier = Modifier.padding(top = 2.dp))
            }
        }
    }
}
