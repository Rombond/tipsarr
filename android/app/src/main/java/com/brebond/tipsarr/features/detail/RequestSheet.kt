package com.brebond.tipsarr.features.detail

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.defaultMinSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.outlined.Add
import androidx.compose.material.icons.outlined.CheckCircle
import androidx.compose.material.icons.outlined.Circle
import androidx.compose.material.icons.outlined.KeyboardArrowDown
import androidx.compose.material.icons.outlined.Storage
import androidx.compose.material.icons.outlined.Tune
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.selected
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.brebond.tipsarr.R
import com.brebond.tipsarr.core.api.ApiError
import com.brebond.tipsarr.core.api.MediaType
import com.brebond.tipsarr.core.api.RequestOptions
import com.brebond.tipsarr.core.api.RequestRecord
import com.brebond.tipsarr.core.images.RemoteImage
import com.brebond.tipsarr.core.images.TmdbSize
import com.brebond.tipsarr.core.support.plural
import com.brebond.tipsarr.design.Tokens
import com.brebond.tipsarr.design.palette
import com.brebond.tipsarr.ui.Banner
import com.brebond.tipsarr.ui.BannerKind
import com.brebond.tipsarr.ui.ButtonKind
import com.brebond.tipsarr.ui.StatusBadge
import com.brebond.tipsarr.ui.TipsarrButton
import com.brebond.tipsarr.ui.TipsarrSheet
import kotlinx.coroutines.launch

/** Movie: quality profile (+ folder). Show: seasons, quality profile (+ folder). */
@Composable
fun RequestSheet(model: MediaDetailModel, onDismiss: () -> Unit, onDone: (RequestRecord) -> Unit) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val detail = model.detail
    val isTv = model.route.type == MediaType.Tv
    val selectable = model.regularSeasons.filter { it.number !in model.coveredSeasons }
    var selected by remember { mutableStateOf(selectable.map { it.number }.toSet()) }
    var profileId by remember { mutableStateOf(model.options?.qualityProfileId) }
    var folder by remember { mutableStateOf(model.options?.let { it.rootFolder.ifEmpty { it.rootFolders.firstOrNull()?.path } }) }
    var errorText by remember { mutableStateOf<String?>(null) }
    val allSelected = selectable.isNotEmpty() && selected.size == selectable.size

    fun submit() {
        errorText = null
        scope.launch {
            try {
                val record = model.submitRequest(
                    seasons = if (isTv) selected.sorted() else null,
                    profileId = if (model.options == null) null else profileId,
                    folder = if (model.canChooseFolder) folder else null,
                )
                onDone(record)
            } catch (e: ApiError) { errorText = e.message(context) }
        }
    }

    TipsarrSheet(onDismiss, fullHeight = true) {
        Column(
            Modifier.fillMaxWidth().verticalScroll(rememberScrollState()).navigationBarsPadding().padding(horizontal = Tokens.Spacing.xl).padding(bottom = Tokens.Spacing.xl),
            verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.xl),
        ) {
            Row(horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.md), verticalAlignment = Alignment.CenterVertically) {
                Box(Modifier.width(56.dp).height(56.dp * Tokens.Size.posterRatio).clip(RoundedCornerShape(Tokens.Radius.sm)).background(Tokens.palette.muted)) {
                    RemoteImage(detail?.posterPath, TmdbSize.W185, Modifier.size(56.dp, 56.dp * Tokens.Size.posterRatio))
                }
                Column(verticalArrangement = Arrangement.spacedBy(2.dp)) {
                    Text(model.route.title, fontSize = Tokens.FontSize.headline, fontWeight = FontWeight.SemiBold, color = Tokens.palette.fg, maxLines = 2, overflow = TextOverflow.Ellipsis)
                    Text(
                        listOfNotNull(detail?.year, stringResource(if (isTv) R.string.type_tv else R.string.type_movie)).joinToString(" · "),
                        fontSize = Tokens.FontSize.subhead, color = Tokens.palette.mutedFg,
                    )
                }
            }
            if (isTv) {
                Column(verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.md)) {
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Text(stringResource(R.string.req_which_seasons), fontSize = Tokens.FontSize.title3, fontWeight = FontWeight.SemiBold, color = Tokens.palette.fg, modifier = Modifier.weight(1f))
                        TextButton({ selected = if (allSelected) emptySet() else selectable.map { it.number }.toSet() }, enabled = selectable.isNotEmpty()) {
                            Text(stringResource(if (allSelected) R.string.req_select_none else R.string.req_select_all))
                        }
                    }
                    model.regularSeasons.forEach { season ->
                        val covered = season.number in model.coveredSeasons
                        val isSelected = season.number in selected
                        Row(
                            Modifier.fillMaxWidth().defaultMinSize(minHeight = Tokens.Size.touchTarget + Tokens.Spacing.md)
                                .clip(RoundedCornerShape(Tokens.Radius.md)).background(Tokens.palette.muted).alpha(if (covered) 0.6f else 1f)
                                .clickable(enabled = !covered) { selected = if (isSelected) selected - season.number else selected + season.number }
                                .semantics { this.selected = isSelected }.padding(Tokens.Spacing.md),
                            horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.md), verticalAlignment = Alignment.CenterVertically,
                        ) {
                            Icon(
                                if (isSelected || covered) Icons.Filled.CheckCircle else Icons.Outlined.Circle, null,
                                tint = if (covered) Tokens.palette.mutedFg else Tokens.palette.fg,
                            )
                            Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
                                Text(season.name, fontSize = Tokens.FontSize.body, fontWeight = FontWeight.SemiBold, color = Tokens.palette.fg)
                                Text(context.plural("seasons_episodes", season.episodeCount), fontSize = Tokens.FontSize.footnote, color = Tokens.palette.mutedFg)
                            }
                            if (covered) StatusBadge(model.request?.state ?: com.brebond.tipsarr.ui.RequestState.Requested)
                        }
                    }
                }
            }
            model.options?.let { options ->
                Column(verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm)) {
                    OptionRow(Icons.Outlined.Tune, stringResource(R.string.req_quality_profile), options.profiles.firstOrNull { it.id == profileId }?.name.orEmpty(),
                        options.profiles.map { it.name to { profileId = it.id } })
                    if (model.canChooseFolder && options.rootFolders.isNotEmpty()) {
                        OptionRow(Icons.Outlined.Storage, stringResource(R.string.req_root_folder), folder ?: options.rootFolder,
                            options.rootFolders.map { "${it.path} · ${formatBytes(it.freeSpace)}" to { folder = it.path } })
                    }
                }
            }
            Text(stringResource(if (model.isAdmin) R.string.m_request_admin_note else R.string.m_request_user_note), fontSize = Tokens.FontSize.footnote, color = Tokens.palette.mutedFg)
            errorText?.let { Banner(it, kind = BannerKind.Error) }
            Column(verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm)) {
                TipsarrButton(
                    stringResource(if (model.busy) R.string.media_requesting else R.string.media_request), ::submit,
                    fullWidth = true, enabled = !model.busy && !(isTv && selected.isEmpty()),
                ) { Icon(Icons.Outlined.Add, null, Modifier.size(20.dp)); Spacer(Modifier.width(Tokens.Spacing.sm)) }
                TipsarrButton(stringResource(R.string.common_cancel), onDismiss, kind = ButtonKind.Ghost, fullWidth = true)
            }
        }
    }
}

internal fun formatBytes(bytes: Long): String {
    val gb = bytes / 1_000_000_000.0
    return if (gb >= 1000) String.format(java.util.Locale.getDefault(), "%.1f TB", gb / 1000) else String.format(java.util.Locale.getDefault(), "%.0f GB", gb)
}

@Composable
private fun OptionRow(icon: ImageVector, title: String, value: String, choices: List<Pair<String, () -> Unit>>) {
    var menu by remember { mutableStateOf(false) }
    Box {
        Row(
            Modifier.fillMaxWidth().defaultMinSize(minHeight = Tokens.Size.touchTarget + Tokens.Spacing.md)
                .clip(RoundedCornerShape(Tokens.Radius.md)).background(Tokens.palette.muted)
                .clickable { menu = true }.padding(horizontal = Tokens.Spacing.md),
            horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.md), verticalAlignment = Alignment.CenterVertically,
        ) {
            Icon(icon, null, tint = Tokens.palette.mutedFg, modifier = Modifier.size(24.dp))
            Text(title, fontSize = Tokens.FontSize.body, color = Tokens.palette.fg, maxLines = 1)
            Text(value, fontSize = Tokens.FontSize.body, color = Tokens.palette.mutedFg, maxLines = 1, overflow = TextOverflow.MiddleEllipsis, modifier = Modifier.weight(1f), textAlign = androidx.compose.ui.text.style.TextAlign.End)
            Icon(Icons.Outlined.KeyboardArrowDown, null, tint = Tokens.palette.mutedFg)
        }
        DropdownMenu(menu, { menu = false }, containerColor = Tokens.palette.card) {
            choices.forEach { (label, pick) -> DropdownMenuItem(text = { Text(label, color = Tokens.palette.fg) }, onClick = { menu = false; pick() }) }
        }
    }
}
