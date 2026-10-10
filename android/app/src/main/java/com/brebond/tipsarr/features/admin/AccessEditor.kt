package com.brebond.tipsarr.features.admin

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Checkbox
import androidx.compose.material3.CheckboxDefaults
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.heading
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import com.brebond.tipsarr.R
import com.brebond.tipsarr.core.api.ApiError
import com.brebond.tipsarr.core.api.Permission
import com.brebond.tipsarr.core.api.PermissionDefaults
import com.brebond.tipsarr.core.api.PermissionFlag
import com.brebond.tipsarr.core.api.Profile
import com.brebond.tipsarr.core.api.TipsarrApi
import com.brebond.tipsarr.core.api.UserQuota
import com.brebond.tipsarr.core.api.setPermissionDefaults
import com.brebond.tipsarr.core.api.updateAccess
import com.brebond.tipsarr.core.support.LocalToast
import com.brebond.tipsarr.core.support.ToastKind
import com.brebond.tipsarr.core.support.stringByName
import com.brebond.tipsarr.design.Tokens
import com.brebond.tipsarr.design.palette
import com.brebond.tipsarr.ui.Banner
import com.brebond.tipsarr.ui.BannerKind
import com.brebond.tipsarr.ui.ButtonKind
import com.brebond.tipsarr.ui.Field
import com.brebond.tipsarr.ui.TipsarrButton
import com.brebond.tipsarr.ui.TipsarrSheet
import kotlinx.coroutines.launch

/** The four request limit fields as text: an empty field means "use the default", 0 means no limit. */
data class QuotaText(val movieLimit: String = "", val movieDays: String = "", val tvLimit: String = "", val tvDays: String = "") {
    fun toQuota() = UserQuota(movieLimit.toIntOrNull(), movieDays.toIntOrNull(), tvLimit.toIntOrNull(), tvDays.toIntOrNull())

    companion object {
        fun of(user: Profile) = QuotaText(
            user.movieQuotaLimit?.toString().orEmpty(), user.movieQuotaDays?.toString().orEmpty(),
            user.tvQuotaLimit?.toString().orEmpty(), user.tvQuotaDays?.toString().orEmpty(),
        )

        fun of(defaults: PermissionDefaults) = QuotaText(
            defaults.movieQuotaLimit.toString(), defaults.movieQuotaDays.toString(), defaults.tvQuotaLimit.toString(), defaults.tvQuotaDays.toString(),
        )
    }
}

private val GROUPS = listOf("requests", "auto-approve", "issues", "admin")

/**
 * The checkbox list of permission flags. Like in Seerr, a flag that brings others with it (manage requests brings
 * auto-approve, request brings movies and TV shows ...) switches them on and keeps them on while it is on.
 * Flags marked `reserved` exist only so the numbers match Seerr and are not shown.
 */
@Composable
fun PermissionEditor(flags: List<PermissionFlag>, mask: Long, onChange: (Long) -> Unit, readOnly: Boolean = false, keysLocked: Boolean = false) {
    val context = LocalContext.current
    val shown = flags.filter { !it.reserved }
    val forced = Permission.forced(mask, flags)
    Column(verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.lg)) {
        GROUPS.forEach { group ->
            val items = shown.filter { it.group == group }
            if (items.isEmpty()) return@forEach
            Column(verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm)) {
                Text(
                    context.stringByName("perm_group_${group.replace('-', '_')}").orEmpty(), fontSize = Tokens.FontSize.footnote, fontWeight = FontWeight.SemiBold,
                    color = Tokens.palette.mutedFg, modifier = Modifier.semantics { heading() },
                )
                items.forEach { flag ->
                    val on = Permission.has(mask, flag.value)
                    val isKey = flag.value == Permission.MANAGE_USERS || flag.value == Permission.MANAGE_SETTINGS
                    val locked = readOnly || (on && Permission.has(forced, flag.value)) || (keysLocked && isKey)
                    Row(verticalAlignment = Alignment.Top, horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm)) {
                        Checkbox(
                            checked = on, enabled = !locked,
                            onCheckedChange = { checked ->
                                val next = if (checked) mask or flag.value else mask and flag.value.inv()
                                onChange(Permission.normalize(next, flags))
                            },
                            colors = CheckboxDefaults.colors(checkedColor = Tokens.palette.primary, checkmarkColor = Tokens.palette.primaryFg),
                        )
                        Column(Modifier.padding(top = 12.dp), verticalArrangement = Arrangement.spacedBy(2.dp)) {
                            Text(context.stringByName("perm_${flag.name}").orEmpty(), fontSize = Tokens.FontSize.body, fontWeight = FontWeight.Medium, color = Tokens.palette.fg)
                            Text(context.stringByName("perm_${flag.name}_desc").orEmpty(), fontSize = Tokens.FontSize.footnote, color = Tokens.palette.mutedFg)
                        }
                    }
                }
            }
        }
    }
}

/** Request limits: movies and seasons of shows per number of days. Placeholders show what applies when a field is empty. */
@Composable
fun QuotaFields(values: QuotaText, onChange: (QuotaText) -> Unit, defaults: PermissionDefaults? = null, readOnly: Boolean = false) {
    fun field(title: String, value: String, placeholder: Int?, set: (String) -> Unit, modifier: Modifier) = @Composable {
        Field(title, value, { if (!readOnly) set(it.filter(Char::isDigit).take(6)) }, modifier, placeholder = placeholder?.toString(), keyboardType = KeyboardType.Number)
    }
    Column(verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.md)) {
        Text(stringResource(R.string.users_quota_title), fontSize = Tokens.FontSize.footnote, fontWeight = FontWeight.SemiBold, color = Tokens.palette.mutedFg, modifier = Modifier.semantics { heading() })
        Text(stringResource(R.string.users_quota_movies), fontSize = Tokens.FontSize.body, fontWeight = FontWeight.Medium, color = Tokens.palette.fg)
        Row(horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.md)) {
            field(stringResource(R.string.users_quota_limit), values.movieLimit, defaults?.movieQuotaLimit, { onChange(values.copy(movieLimit = it)) }, Modifier.weight(1f))()
            field(stringResource(R.string.users_quota_days), values.movieDays, defaults?.movieQuotaDays, { onChange(values.copy(movieDays = it)) }, Modifier.weight(1f))()
        }
        Text(stringResource(R.string.users_quota_shows), fontSize = Tokens.FontSize.body, fontWeight = FontWeight.Medium, color = Tokens.palette.fg)
        Row(horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.md)) {
            field(stringResource(R.string.users_quota_limit), values.tvLimit, defaults?.tvQuotaLimit, { onChange(values.copy(tvLimit = it)) }, Modifier.weight(1f))()
            field(stringResource(R.string.users_quota_days), values.tvDays, defaults?.tvQuotaDays, { onChange(values.copy(tvDays = it)) }, Modifier.weight(1f))()
        }
        Text(stringResource(R.string.users_quota_hint), fontSize = Tokens.FontSize.footnote, color = Tokens.palette.mutedFg)
    }
}

/** Permissions and request limits of one person. Read-only for admins (they can do everything) and for your own account. */
@Composable
fun UserAccessSheet(
    api: TipsarrApi,
    user: Profile,
    me: Profile,
    flags: List<PermissionFlag>,
    defaults: PermissionDefaults,
    onDismiss: () -> Unit,
    onSaved: (Profile) -> Unit,
) {
    val context = LocalContext.current
    val toast = LocalToast.current
    val scope = rememberCoroutineScope()
    var mask by remember { mutableStateOf(user.permissions) }
    var quota by remember { mutableStateOf(QuotaText.of(user)) }
    var busy by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    val readOnly = user.isAdmin || user.id == me.id

    TipsarrSheet(onDismiss, fullHeight = true) {
        Column(
            Modifier.fillMaxWidth().verticalScroll(rememberScrollState()).navigationBarsPadding().padding(horizontal = Tokens.Spacing.xl).padding(bottom = Tokens.Spacing.xl),
            verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.xl),
        ) {
            Text(stringResource(R.string.users_permissions_title, user.name), fontSize = Tokens.FontSize.title3, fontWeight = FontWeight.SemiBold, color = Tokens.palette.fg, modifier = Modifier.semantics { heading() })
            val note = when {
                user.isAdmin -> R.string.users_permissions_admin
                user.id == me.id -> R.string.users_permissions_own
                !me.isAdmin -> R.string.users_permissions_keys
                else -> null
            }
            note?.let { Text(stringResource(it), fontSize = Tokens.FontSize.footnote, color = Tokens.palette.mutedFg) }
            PermissionEditor(flags, mask, { mask = it }, readOnly = readOnly, keysLocked = !me.isAdmin)
            QuotaFields(quota, { quota = it }, defaults = defaults, readOnly = readOnly)
            error?.let { Banner(it, kind = BannerKind.Error) }
            Column(verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm)) {
                TipsarrButton(
                    stringResource(if (busy) R.string.common_saving else R.string.common_save),
                    {
                        busy = true; error = null
                        scope.launch {
                            try {
                                onSaved(api.updateAccess(user.id, mask, quota.toQuota()))
                                toast?.show(context.getString(R.string.users_permissions_saved))
                            } catch (e: ApiError) { error = e.message(context) } finally { busy = false }
                        }
                    },
                    fullWidth = true, enabled = !busy && !readOnly,
                )
                TipsarrButton(stringResource(R.string.common_cancel), onDismiss, kind = ButtonKind.Ghost, fullWidth = true)
            }
        }
    }
}

/** What new people get when they first sign in. Everyone who manages users sees it; only admins can change it. */
@Composable
fun PermissionDefaultsCard(api: TipsarrApi, me: Profile, flags: List<PermissionFlag>, defaults: PermissionDefaults, onSaved: (PermissionDefaults) -> Unit) {
    val context = LocalContext.current
    val toast = LocalToast.current
    val scope = rememberCoroutineScope()
    var mask by remember(defaults) { mutableStateOf(defaults.permissions) }
    var quota by remember(defaults) { mutableStateOf(QuotaText.of(defaults)) }
    var busy by remember { mutableStateOf(false) }
    Column(Modifier.fillMaxWidth(), verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.lg)) {
        Text(stringResource(R.string.users_defaults_title), fontSize = Tokens.FontSize.title3, fontWeight = FontWeight.Bold, color = Tokens.palette.fg, modifier = Modifier.semantics { heading() })
        Text(stringResource(R.string.users_defaults_desc), fontSize = Tokens.FontSize.footnote, color = Tokens.palette.mutedFg)
        PermissionEditor(flags, mask, { mask = it }, readOnly = !me.isAdmin)
        QuotaFields(quota, { quota = it }, readOnly = !me.isAdmin)
        if (me.isAdmin) {
            TipsarrButton(
                stringResource(if (busy) R.string.common_saving else R.string.common_save),
                {
                    busy = true
                    scope.launch {
                        try {
                            fun n(s: String) = s.toIntOrNull() ?: 0
                            onSaved(api.setPermissionDefaults(PermissionDefaults(mask, n(quota.movieLimit), n(quota.movieDays), n(quota.tvLimit), n(quota.tvDays))))
                            toast?.show(context.getString(R.string.common_saved))
                        } catch (e: ApiError) { toast?.show(e.message(context), ToastKind.Error) } finally { busy = false }
                    }
                },
                fullWidth = true, enabled = !busy,
            )
        }
        Spacer(Modifier.padding(bottom = Tokens.Spacing.sm))
    }
}
