package com.brebond.tipsarr.features.admin

import androidx.annotation.StringRes
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.itemsIndexed
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.heading
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.brebond.tipsarr.R
import com.brebond.tipsarr.core.api.LibraryItem
import com.brebond.tipsarr.core.api.RequestRecord
import com.brebond.tipsarr.core.api.StatsTop
import com.brebond.tipsarr.core.images.RemoteImage
import com.brebond.tipsarr.core.images.TmdbSize
import com.brebond.tipsarr.design.Tokens
import com.brebond.tipsarr.design.palette
import com.brebond.tipsarr.features.detail.MediaRoute
import com.brebond.tipsarr.ui.TopBarScaffold
import java.text.DateFormat
import java.util.Date
import java.util.Locale

/** One title in a carousel or a list: a poster, a title and a short note. */
data class PosterRow(
    val id: String,
    val title: String,
    val note: String,
    val posterPath: String?,
    /** True when `posterPath` is a path on the Tipsarr server (Jellyfin posters) instead of a TMDB path. */
    val onServer: Boolean,
    val route: MediaRoute,
)

/** Carries a whole list to its full-screen view. */
data class PosterListRoute(@StringRes val title: Int, val rows: List<PosterRow>, val ranked: Boolean = false)

/** A row of posters in a carousel with a See all button. */
@Composable
fun PosterCarousel(
    @StringRes title: Int,
    rows: List<PosterRow>,
    onOpen: (MediaRoute) -> Unit,
    onSeeAll: (PosterListRoute) -> Unit,
    modifier: Modifier = Modifier,
    ranked: Boolean = false,
    horizontalPadding: androidx.compose.ui.unit.Dp = 0.dp,
) {
    if (rows.isEmpty()) return
    Column(modifier, verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.md)) {
        Row(Modifier.padding(horizontal = horizontalPadding), verticalAlignment = Alignment.CenterVertically) {
            Text(stringResource(title), fontSize = Tokens.FontSize.title3, fontWeight = FontWeight.Bold, color = Tokens.palette.fg, modifier = Modifier.weight(1f).semantics { heading() })
            Text(
                stringResource(R.string.m_discover_see_all), fontSize = Tokens.FontSize.subhead, color = Tokens.palette.mutedFg,
                modifier = Modifier.clickable { onSeeAll(PosterListRoute(title, rows, ranked)) }.padding(Tokens.Spacing.sm),
            )
        }
        LazyRow(horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.md), contentPadding = PaddingValues(horizontal = horizontalPadding)) {
            items(rows, key = { it.id }) { row ->
                Column(Modifier.width(120.dp).clickable { onOpen(row.route) }, verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.xs)) {
                    Box(Modifier.fillMaxWidth().height(180.dp).clip(RoundedCornerShape(Tokens.Radius.md)).background(Tokens.palette.muted)) {
                        RemoteImage(row.posterPath, TmdbSize.W342, Modifier.fillMaxSize(), server = row.onServer)
                    }
                    Text(row.title, fontSize = Tokens.FontSize.subhead, fontWeight = FontWeight.Medium, color = Tokens.palette.fg, maxLines = 1, overflow = TextOverflow.Ellipsis)
                    Text(row.note, fontSize = Tokens.FontSize.caption, color = Tokens.palette.mutedFg, maxLines = 1, overflow = TextOverflow.Ellipsis)
                }
            }
        }
    }
}

/** The same list as rows. */
@Composable
fun PosterListScreen(route: PosterListRoute, onBack: () -> Unit, onOpen: (MediaRoute) -> Unit) {
    TopBarScaffold(stringResource(route.title), onBack) {
        LazyColumn(Modifier.fillMaxSize(), contentPadding = PaddingValues(horizontal = Tokens.Spacing.lg)) {
            itemsIndexed(route.rows, key = { _, row -> row.id }) { index, row ->
                Row(
                    Modifier.fillMaxWidth().clickable { onOpen(row.route) }.padding(vertical = Tokens.Spacing.sm),
                    horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.md), verticalAlignment = Alignment.CenterVertically,
                ) {
                    if (route.ranked) Text((index + 1).toString(), fontSize = Tokens.FontSize.headline, color = Tokens.palette.mutedFg, modifier = Modifier.width(28.dp))
                    Box(Modifier.width(44.dp).height(66.dp).clip(RoundedCornerShape(6.dp)).background(Tokens.palette.muted)) {
                        RemoteImage(row.posterPath, TmdbSize.W92, Modifier.fillMaxSize(), server = row.onServer)
                    }
                    Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
                        Text(row.title, fontSize = Tokens.FontSize.body, fontWeight = FontWeight.Medium, color = Tokens.palette.fg, maxLines = 2, overflow = TextOverflow.Ellipsis)
                        if (row.note.isNotEmpty()) Text(row.note, fontSize = Tokens.FontSize.footnote, color = Tokens.palette.mutedFg)
                    }
                }
                HorizontalDivider(color = Tokens.palette.border)
            }
        }
    }
}

object StatsFormat {
    fun hours(value: Double, unit: String): String =
        if (value >= 100) "${Math.round(value)} $unit" else String.format(Locale.getDefault(), "%.1f", value) + " $unit"
}

fun StatsTop.toRow(context: android.content.Context): PosterRow {
    val unit = context.getString(R.string.stats_unit_h)
    val count = context.getString(if (type == com.brebond.tipsarr.core.api.MediaType.Tv) R.string.stats_n_episodes else R.string.stats_n_plays, plays.toString())
    return PosterRow("$type-$tmdbId", title, "${StatsFormat.hours(hours, unit)} · $count", posterPath, true, MediaRoute(type, tmdbId, title))
}

/** "Asked for, never watched": since when the title is available for the requester. */
fun RequestRecord.toUnwatchedRow(context: android.content.Context): PosterRow = PosterRow(
    id, title, context.getString(R.string.stats_unwatched_since, DateFormat.getDateInstance(DateFormat.MEDIUM).format(Date(createdAt * 1000))),
    posterPath, false, MediaRoute(type, tmdbId, title),
)

fun LibraryItem.toRow(): PosterRow = PosterRow(id, title, year?.toString().orEmpty(), posterPath, true, MediaRoute(type, tmdbId, title))
