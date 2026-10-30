package com.brebond.tipsarr

import android.content.Context
import android.Manifest
import android.os.Bundle
import androidx.activity.result.contract.ActivityResultContracts
import kotlinx.coroutines.CompletableDeferred
import com.brebond.tipsarr.core.push.PushManager
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

    private var permissionAnswer: CompletableDeferred<Boolean>? = null
    private val notificationPermission = registerForActivityResult(ActivityResultContracts.RequestPermission()) { permissionAnswer?.complete(it) }

    /** A tapped alert (the system's own, while the app was closed, or ours) carries the event, id and server as extras. */
    private fun handlePush(intent: android.content.Intent) {
        val event = intent.getStringExtra("event") ?: return
        val id = intent.getStringExtra("id") ?: return
        intent.removeExtra("event")
        viewModel.handlePush(event, id, intent.getStringExtra("server"))
    }

    override fun onNewIntent(intent: android.content.Intent) {
        super.onNewIntent(intent)
        intent.data?.let(viewModel::handle)
        handlePush(intent)
    }

    override fun onDestroy() {
        PushManager.permissionRequester = null
        super.onDestroy()
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        intent?.data?.let(viewModel::handle)
        if (savedInstanceState == null) intent?.let(::handlePush)
        PushManager.permissionRequester = {
            CompletableDeferred<Boolean>().also { permissionAnswer = it; notificationPermission.launch(Manifest.permission.POST_NOTIFICATIONS) }.await()
        }
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
