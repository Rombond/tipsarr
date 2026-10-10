package com.brebond.tipsarr

import android.content.Context
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.activity.viewModels
import androidx.compose.foundation.isSystemInDarkTheme
import com.brebond.tipsarr.core.support.AppSettings
import com.brebond.tipsarr.core.support.AppTheme
import com.brebond.tipsarr.design.TipsarrTheme

class MainActivity : ComponentActivity() {
    private val viewModel: SessionViewModel by viewModels()
    private val settings by lazy { AppSettings(this) }

    /** The language picked in Settings (null follows the phone) applies to every resource lookup. */
    override fun attachBaseContext(newBase: Context) = super.attachBaseContext(AppSettings.wrap(newBase))

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        setContent {
            val dark = when (settings.theme) {
                AppTheme.System -> isSystemInDarkTheme()
                AppTheme.Light -> false
                AppTheme.Dark -> true
            }
            TipsarrTheme(dark = dark) {
                Root(viewModel.session, settings, onLanguageChanged = { recreate() })
            }
        }
    }
}
