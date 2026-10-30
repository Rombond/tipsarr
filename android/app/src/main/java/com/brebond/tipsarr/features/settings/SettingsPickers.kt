package com.brebond.tipsarr.features.settings

import android.content.ComponentName
import android.content.Context
import android.content.pm.PackageManager
import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.defaultMinSize
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.Check
import androidx.compose.material.icons.outlined.Search
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.text.TextStyle
import com.brebond.tipsarr.R
import com.brebond.tipsarr.core.api.RatingSource
import com.brebond.tipsarr.design.Tokens
import com.brebond.tipsarr.design.palette
import com.brebond.tipsarr.ui.ProviderMark
import com.brebond.tipsarr.ui.TopBarScaffold
import java.util.Locale

/** Countries sorted by their name in the current language: code and "flag name". */
internal fun regionChoices(): List<Pair<String, String>> {
    val locale = Locale.getDefault()
    // Sorted on the name alone: with the flag in front the order followed the country code (Andorra before Afghanistan).
    val collator = java.text.Collator.getInstance(locale)
    return Locale.getISOCountries().map { it to Locale("", it).getDisplayCountry(locale) }
        .filter { it.second.isNotEmpty() }.sortedWith(compareBy(collator) { it.second })
        .map { (code, name) -> code to flag(code) + " " + name }
}

@Composable
private fun CheckMark() = Icon(Icons.Outlined.Check, null, tint = Tokens.palette.fg)

/** Region: a searchable list of countries, a check on the current one. */
@Composable
fun RegionScreen(selection: String, onPick: (String) -> Unit, onBack: () -> Unit) {
    val regions = remember { regionChoices() }
    var query by remember { mutableStateOf("") }
    val shown = remember(query) {
        if (query.isBlank()) regions else regions.filter { it.second.contains(query, ignoreCase = true) || it.first.contains(query, ignoreCase = true) }
    }
    TopBarScaffold(stringResource(R.string.m_settings_region), onBack) {
        Column(Modifier.fillMaxSize()) {
            Row(
                Modifier.fillMaxWidth().padding(horizontal = Tokens.Spacing.lg, vertical = Tokens.Spacing.sm).clip(RoundedCornerShape(Tokens.Radius.md))
                    .background(Tokens.palette.muted).defaultMinSize(minHeight = Tokens.Size.touchTarget).padding(horizontal = Tokens.Spacing.md),
                verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm),
            ) {
                Icon(Icons.Outlined.Search, null, tint = Tokens.palette.mutedFg)
                Box(Modifier.weight(1f)) {
                    if (query.isEmpty()) Text(stringResource(R.string.m_settings_region), fontSize = Tokens.FontSize.body, color = Tokens.palette.mutedFg)
                    BasicTextField(
                        query, { query = it }, singleLine = true, modifier = Modifier.fillMaxWidth(),
                        textStyle = TextStyle(color = Tokens.palette.fg, fontSize = Tokens.FontSize.body),
                        cursorBrush = SolidColor(Tokens.palette.fg), keyboardOptions = KeyboardOptions.Default,
                    )
                }
            }
            LazyColumn(Modifier.fillMaxSize()) {
                items(shown, key = { it.first }) { (code, label) ->
                    Row(
                        Modifier.fillMaxWidth().defaultMinSize(minHeight = Tokens.Size.touchTarget + Tokens.Spacing.md).clickable { onPick(code) }.padding(horizontal = Tokens.Spacing.lg),
                        verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.md),
                    ) {
                        Text(label, fontSize = Tokens.FontSize.body, color = Tokens.palette.fg, modifier = Modifier.weight(1f))
                        if (code == selection) CheckMark()
                    }
                }
            }
        }
    }
}

@Composable
internal fun ratingSourceTitle(source: RatingSource): String = stringResource(
    when (source) {
        RatingSource.Tmdb -> R.string.m_settings_rating_tmdb
        RatingSource.Imdb -> R.string.m_settings_rating_imdb
        RatingSource.Metacritic -> R.string.m_settings_rating_metacritic
        RatingSource.RottenTomatoes -> R.string.m_settings_rating_rt
    },
)

/** The score shown on posters: each choice with its logo. */
@Composable
fun RatingSourceScreen(selection: RatingSource, onPick: (RatingSource) -> Unit, onBack: () -> Unit) {
    TopBarScaffold(stringResource(R.string.profile_rating_source), onBack) {
        Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
            RatingSource.entries.forEach { source ->
                Row(
                    Modifier.fillMaxWidth().defaultMinSize(minHeight = Tokens.Size.touchTarget + Tokens.Spacing.md).clickable { onPick(source) }.padding(horizontal = Tokens.Spacing.lg),
                    verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.md),
                ) {
                    Box(Modifier.width(56.dp), contentAlignment = Alignment.CenterStart) { ProviderMark(source, value = 80.0, height = 18.dp) }
                    Text(ratingSourceTitle(source), fontSize = Tokens.FontSize.body, color = Tokens.palette.fg, modifier = Modifier.weight(1f))
                    if (source == selection) CheckMark()
                }
            }
        }
    }
}

/** App icons the person can choose from: one launcher alias of the manifest each. */
enum class AppIconChoice(val alias: String, val preview: Int, val title: Int) {
    Light("IconLight", R.drawable.ic_icon_light, R.string.m_icon_light),
    Dark("IconDark", R.drawable.ic_icon_dark, R.string.m_icon_dark),
    Mono("IconMono", R.drawable.ic_icon_mono, R.string.m_icon_mono);

    companion object {
        private fun component(context: Context, choice: AppIconChoice) = ComponentName(context.packageName, "${context.packageName}.${choice.alias}")

        fun current(context: Context): AppIconChoice = entries.firstOrNull {
            val state = context.packageManager.getComponentEnabledSetting(component(context, it))
            state == PackageManager.COMPONENT_ENABLED_STATE_ENABLED || (state == PackageManager.COMPONENT_ENABLED_STATE_DEFAULT && it == Light)
        } ?: Light

        /** Enables the new alias first so the launcher never has none. */
        fun select(context: Context, choice: AppIconChoice) {
            val pm = context.packageManager
            pm.setComponentEnabledSetting(component(context, choice), PackageManager.COMPONENT_ENABLED_STATE_ENABLED, PackageManager.DONT_KILL_APP)
            entries.filter { it != choice }.forEach {
                pm.setComponentEnabledSetting(component(context, it), PackageManager.COMPONENT_ENABLED_STATE_DISABLED, PackageManager.DONT_KILL_APP)
            }
        }
    }
}

@Composable
fun AppIconScreen(onBack: () -> Unit) {
    val context = LocalContext.current
    var current by remember { mutableStateOf(AppIconChoice.current(context)) }
    TopBarScaffold(stringResource(R.string.m_settings_app_icon), onBack) {
        Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState()).padding(Tokens.Spacing.lg), verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm)) {
            Column(Modifier.fillMaxWidth().clip(RoundedCornerShape(Tokens.Radius.md)).background(Tokens.palette.card)) {
                AppIconChoice.entries.forEach { choice ->
                    Row(
                        Modifier.fillMaxWidth().defaultMinSize(minHeight = 64.dp).clickable {
                            if (choice != current) { AppIconChoice.select(context, choice); current = choice }
                        }.padding(horizontal = Tokens.Spacing.md, vertical = Tokens.Spacing.sm),
                        verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.md),
                    ) {
                        Box(Modifier.size(48.dp).clip(RoundedCornerShape(11.dp)).background(Tokens.palette.bg).border(BorderStroke(1.dp, Tokens.palette.border), RoundedCornerShape(11.dp))) {
                            Image(painterResource(choice.preview), null, Modifier.fillMaxSize())
                        }
                        Text(stringResource(choice.title), fontSize = Tokens.FontSize.body, color = Tokens.palette.fg, modifier = Modifier.weight(1f))
                        if (current == choice) CheckMark()
                    }
                }
            }
            Text(stringResource(R.string.m_icon_footer), fontSize = Tokens.FontSize.footnote, color = Tokens.palette.mutedFg, textAlign = TextAlign.Start, modifier = Modifier.padding(horizontal = Tokens.Spacing.xs))
        }
    }
}
