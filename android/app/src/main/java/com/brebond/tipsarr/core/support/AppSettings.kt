package com.brebond.tipsarr.core.support

import android.content.Context
import android.content.res.Configuration
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import java.util.Locale

enum class AppTheme { System, Light, Dark }

/** Language the person picked in Settings (null follows the phone) and the theme, kept in preferences. */
class AppSettings(context: Context) {
    private val prefs = context.applicationContext.getSharedPreferences(PREFS, Context.MODE_PRIVATE)

    var theme by mutableStateOf(AppTheme.entries.firstOrNull { it.name == prefs.getString(THEME, null) } ?: AppTheme.System)
        private set

    var language by mutableStateOf(prefs.getString(LANGUAGE, null))
        private set

    /** Teasers sound: off once the person muted a trailer, until they turn it back on. Sound is on by default. */
    var teaserMuted by mutableStateOf(prefs.getBoolean(TEASER_MUTED, false))
        private set

    fun updateTeaserMuted(value: Boolean) {
        teaserMuted = value
        prefs.edit().putBoolean(TEASER_MUTED, value).apply()
    }

    fun updateTheme(value: AppTheme) {
        theme = value
        prefs.edit().putString(THEME, value.name).apply()
    }

    fun updateLanguage(value: String?) {
        language = value
        prefs.edit().apply { if (value == null) remove(LANGUAGE) else putString(LANGUAGE, value) }.apply()
    }

    companion object {
        private const val PREFS = "app_settings"
        private const val THEME = "theme"
        private const val LANGUAGE = "language"
        private const val TEASER_MUTED = "teaser_muted"
        /** The languages of locales/languages.json (generated). */
        val supported = GeneratedLanguages.all

        /** Context whose resources use the chosen language; also sets the JVM default so API headers follow it. */
        fun wrap(base: Context): Context {
            val code = base.getSharedPreferences(PREFS, Context.MODE_PRIVATE).getString(LANGUAGE, null) ?: return base
            val locale = Locale.forLanguageTag(code)
            Locale.setDefault(locale)
            val config = Configuration(base.resources.configuration).apply { setLocale(locale) }
            return base.createConfigurationContext(config)
        }
    }
}
