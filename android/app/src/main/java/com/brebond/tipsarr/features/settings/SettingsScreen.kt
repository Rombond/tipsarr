package com.brebond.tipsarr.features.settings

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.defaultMinSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.outlined.KeyboardArrowRight
import androidx.compose.material.icons.automirrored.outlined.Logout
import androidx.compose.material.icons.outlined.Devices
import androidx.compose.material.icons.outlined.Dns
import androidx.compose.material.icons.outlined.Language
import androidx.compose.material.icons.outlined.Palette
import androidx.compose.material.icons.outlined.People
import androidx.compose.material.icons.outlined.Person
import androidx.compose.material.icons.outlined.Public
import androidx.compose.material.icons.outlined.Star
import androidx.compose.material.icons.outlined.VisibilityOff
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import com.brebond.tipsarr.R
import com.brebond.tipsarr.core.api.ApiError
import com.brebond.tipsarr.core.api.Profile
import com.brebond.tipsarr.core.api.RatingSource
import com.brebond.tipsarr.core.api.TipsarrApi
import com.brebond.tipsarr.core.api.sessions
import com.brebond.tipsarr.core.api.updatePreferences
import com.brebond.tipsarr.core.auth.Account
import com.brebond.tipsarr.core.support.AppSettings
import com.brebond.tipsarr.core.support.AppTheme
import com.brebond.tipsarr.core.support.LocalToast
import com.brebond.tipsarr.core.support.ToastKind
import com.brebond.tipsarr.design.Tokens
import com.brebond.tipsarr.design.palette
import com.brebond.tipsarr.ui.TopBarScaffold
import kotlinx.coroutines.launch
import java.util.Locale

/** The country's flag as two regional-indicator symbols. */
private fun flag(code: String): String = code.uppercase().map { String(Character.toChars(0x1F1E6 + (it - 'A'))) }.joinToString("")

/** Settings: account, language, region, score on posters, hidden titles, theme, devices, server, sign out. */
@Composable
fun SettingsScreen(
    account: Account,
    profile: Profile,
    api: TipsarrApi,
    settings: AppSettings,
    accountCount: Int,
    serverVersion: String?,
    onBack: () -> Unit,
    onProfileChanged: (Profile) -> Unit,
    onLanguageChanged: () -> Unit,
    onOpenAccounts: () -> Unit,
    onOpenDevices: () -> Unit,
    onOpenHidden: () -> Unit,
    onChangePicture: () -> Unit,
    onSignOut: () -> Unit,
) {
    val context = LocalContext.current
    val toast = LocalToast.current
    val scope = rememberCoroutineScope()
    var confirmSignOut by remember { mutableStateOf(false) }
    var deviceCount by remember { mutableStateOf<Int?>(null) }
    LaunchedEffect(Unit) { deviceCount = runCatching { api.sessions().size }.getOrNull() }

    fun save(message: Int, work: suspend () -> Profile) {
        scope.launch {
            try {
                onProfileChanged(work())
                toast?.show(context.getString(message))
            } catch (e: ApiError) { toast?.show(e.message(context), ToastKind.Error) }
        }
    }

    val regions = remember(settings.language) {
        val locale = Locale.getDefault()
        Locale.getISOCountries().map { it to flag(it) + " " + Locale("", it).getDisplayCountry(locale) }.filter { it.second.isNotEmpty() }.sortedBy { it.second.lowercase() }
    }

    TopBarScaffold(stringResource(R.string.m_settings_title), onBack) {
        Box(Modifier.fillMaxWidth().verticalScroll(rememberScrollState()).navigationBarsPadding(), contentAlignment = Alignment.TopCenter) {
            Column(Modifier.widthIn(max = 720.dp).fillMaxWidth().padding(Tokens.Spacing.lg), verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.xl)) {
                Group(stringResource(R.string.m_settings_account)) {
                    NavRow(Icons.Outlined.People, stringResource(if (accountCount == 1) R.string.m_accounts_title_one else R.string.m_accounts_title), value = accountCount.toString(), onClick = onOpenAccounts)
                    NavRow(Icons.Outlined.Person, stringResource(R.string.profile_change_picture), onClick = onChangePicture)
                    MenuRow(
                        Icons.Outlined.Language, stringResource(R.string.lang_title),
                        value = AppSettings.supported.firstOrNull { it.first == settings.language }?.second ?: stringResource(R.string.m_settings_language_default),
                        choices = listOf(stringResource(R.string.m_settings_language_default) to null) + AppSettings.supported.map { it.second to it.first },
                    ) { code ->
                        settings.updateLanguage(code)
                        scope.launch {
                            try { onProfileChanged(api.updatePreferences(language = code ?: "")); toast?.show(context.getString(R.string.profile_saved_titles)) } catch (e: ApiError) { toast?.show(e.message(context), ToastKind.Error) }
                            onLanguageChanged()
                        }
                    }
                    MenuRow(
                        Icons.Outlined.Public, stringResource(R.string.m_settings_region),
                        value = regions.firstOrNull { it.first == profile.region.uppercase() }?.second ?: "",
                        choices = regions.map { it.second to it.first },
                    ) { code -> if (code != null) save(R.string.profile_saved) { api.updatePreferences(region = code) } }
                    val sources = listOf(
                        RatingSource.Tmdb to R.string.m_settings_rating_tmdb, RatingSource.Imdb to R.string.m_settings_rating_imdb,
                        RatingSource.Metacritic to R.string.m_settings_rating_metacritic, RatingSource.RottenTomatoes to R.string.m_settings_rating_rt,
                    )
                    val current = RatingSource.from(profile.ratingSource)
                    MenuRow(
                        Icons.Outlined.Star, stringResource(R.string.profile_rating_source),
                        value = stringResource(sources.first { it.first == current }.second),
                        choices = sources.map { stringResource(it.second) to it.first.wire },
                    ) { wire -> if (wire != null) save(R.string.profile_saved) { api.updatePreferences(ratingSource = RatingSource.from(wire)) } }
                    NavRow(Icons.Outlined.VisibilityOff, stringResource(R.string.m_settings_hidden), onClick = onOpenHidden)
                }
                Group(stringResource(R.string.m_settings_appearance)) {
                    val themes = listOf(AppTheme.System to R.string.theme_system, AppTheme.Light to R.string.theme_light, AppTheme.Dark to R.string.theme_dark)
                    MenuRow(
                        Icons.Outlined.Palette, stringResource(R.string.theme_title),
                        value = stringResource(themes.first { it.first == settings.theme }.second),
                        choices = themes.map { stringResource(it.second) to it.first.name },
                    ) { name -> AppTheme.entries.firstOrNull { it.name == name }?.let(settings::updateTheme) }
                }
                Group(stringResource(R.string.m_settings_security)) {
                    NavRow(Icons.Outlined.Devices, stringResource(R.string.profile_devices_title), value = deviceCount?.toString(), onClick = onOpenDevices)
                }
                Group(stringResource(R.string.m_settings_server)) {
                    Row(Modifier.fillMaxWidth().padding(Tokens.Spacing.md), horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.md), verticalAlignment = Alignment.CenterVertically) {
                        Icon(Icons.Outlined.Dns, null, tint = Tokens.palette.mutedFg)
                        Column {
                            Text(account.host, fontSize = Tokens.FontSize.body, color = Tokens.palette.fg)
                            serverVersion?.let { Text(stringResource(R.string.m_settings_server_version, it), fontSize = Tokens.FontSize.footnote, color = Tokens.palette.mutedFg) }
                        }
                    }
                }
                Column(verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm)) {
                    Row(
                        Modifier.fillMaxWidth().defaultMinSize(minHeight = Tokens.Size.touchTarget + Tokens.Spacing.md).clip(RoundedCornerShape(Tokens.Radius.md))
                            .background(Tokens.palette.card).clickable { confirmSignOut = true },
                        horizontalArrangement = Arrangement.Center, verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Icon(Icons.AutoMirrored.Outlined.Logout, null, tint = Tokens.palette.destructive)
                        Spacer(Modifier.width(Tokens.Spacing.sm))
                        Text(stringResource(R.string.m_settings_sign_out), fontSize = Tokens.FontSize.body, fontWeight = FontWeight.Medium, color = Tokens.palette.destructive)
                    }
                    Text(stringResource(R.string.m_accounts_footer_signout), fontSize = Tokens.FontSize.footnote, color = Tokens.palette.mutedFg, textAlign = TextAlign.Center, modifier = Modifier.fillMaxWidth())
                }
            }
        }
    }
    if (confirmSignOut) {
        AlertDialog(
            onDismissRequest = { confirmSignOut = false },
            title = { Text(stringResource(R.string.m_settings_sign_out_confirm)) },
            confirmButton = { TextButton({ confirmSignOut = false; onSignOut() }) { Text(stringResource(R.string.m_settings_sign_out), color = Tokens.palette.destructive) } },
            dismissButton = { TextButton({ confirmSignOut = false }) { Text(stringResource(R.string.common_cancel)) } },
            containerColor = Tokens.palette.card,
        )
    }
}

@Composable
internal fun Group(title: String, content: @Composable () -> Unit) {
    Column(verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm)) {
        Text(title, fontSize = Tokens.FontSize.footnote, fontWeight = FontWeight.SemiBold, color = Tokens.palette.mutedFg, modifier = Modifier.padding(horizontal = Tokens.Spacing.xs))
        Column(Modifier.fillMaxWidth().clip(RoundedCornerShape(Tokens.Radius.md)).background(Tokens.palette.card)) { content() }
    }
}

@Composable
internal fun NavRow(icon: ImageVector, title: String, value: String? = null, onClick: () -> Unit) {
    Row(
        Modifier.fillMaxWidth().defaultMinSize(minHeight = Tokens.Size.touchTarget + Tokens.Spacing.md).clickable(onClick = onClick).padding(horizontal = Tokens.Spacing.md),
        horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.md), verticalAlignment = Alignment.CenterVertically,
    ) {
        Icon(icon, null, tint = Tokens.palette.fg)
        Text(title, fontSize = Tokens.FontSize.body, color = Tokens.palette.fg, modifier = Modifier.weight(1f))
        if (value != null) Text(value, fontSize = Tokens.FontSize.body, color = Tokens.palette.mutedFg)
        Icon(Icons.AutoMirrored.Outlined.KeyboardArrowRight, null, tint = Tokens.palette.mutedFg)
    }
}

/** A row that opens a menu of choices; the chosen value (or null for the "default" entry) goes to `onPick`. */
@Composable
internal fun MenuRow(icon: ImageVector, title: String, value: String, choices: List<Pair<String, String?>>, onPick: (String?) -> Unit) {
    var open by remember { mutableStateOf(false) }
    Box {
        Row(
            Modifier.fillMaxWidth().defaultMinSize(minHeight = Tokens.Size.touchTarget + Tokens.Spacing.md).clickable { open = true }.padding(horizontal = Tokens.Spacing.md),
            horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.md), verticalAlignment = Alignment.CenterVertically,
        ) {
            Icon(icon, null, tint = Tokens.palette.fg)
            Text(title, fontSize = Tokens.FontSize.body, color = Tokens.palette.fg, modifier = Modifier.weight(1f))
            Text(value, fontSize = Tokens.FontSize.body, color = Tokens.palette.mutedFg, maxLines = 1)
        }
        DropdownMenu(open, { open = false }, containerColor = Tokens.palette.card, modifier = Modifier.heightIn(max = 420.dp)) {
            choices.forEach { (label, key) -> DropdownMenuItem(text = { Text(label, color = Tokens.palette.fg) }, onClick = { open = false; onPick(key) }) }
        }
    }
}
