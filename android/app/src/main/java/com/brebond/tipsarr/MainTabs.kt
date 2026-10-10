package com.brebond.tipsarr

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.AccountCircle
import androidx.compose.material.icons.outlined.Checklist
import androidx.compose.material.icons.outlined.Explore
import androidx.compose.material.icons.outlined.Search
import androidx.compose.material.icons.outlined.VideoLibrary
import androidx.compose.material3.Icon
import androidx.compose.material3.NavigationBar
import androidx.compose.material3.NavigationBarItem
import androidx.compose.material3.NavigationBarItemDefaults
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.key
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.activity.compose.BackHandler
import com.brebond.tipsarr.core.api.MediaItem
import com.brebond.tipsarr.core.support.LocalToast
import com.brebond.tipsarr.core.support.ToastCenter
import com.brebond.tipsarr.core.support.ToastHost
import com.brebond.tipsarr.features.detail.MediaDetailScreen
import com.brebond.tipsarr.features.detail.MediaRoute
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.lifecycle.viewmodel.compose.viewModel
import androidx.lifecycle.viewmodel.initializer
import androidx.lifecycle.viewmodel.viewModelFactory
import com.brebond.tipsarr.core.api.Profile
import com.brebond.tipsarr.core.api.TipsarrApi
import com.brebond.tipsarr.core.auth.Account
import com.brebond.tipsarr.core.images.ImageSource
import com.brebond.tipsarr.core.images.LocalAppImageLoader
import com.brebond.tipsarr.core.images.LocalImageSource
import com.brebond.tipsarr.core.images.createImageLoader
import com.brebond.tipsarr.design.Tokens
import com.brebond.tipsarr.design.palette
import com.brebond.tipsarr.features.discover.DiscoverScreen
import com.brebond.tipsarr.ui.StateView

/** Same order as the iOS app: Discover, Requests, Library, Profile, Search (Search is the last tab). */
enum class AppTab(val title: Int, val icon: ImageVector) {
    Discover(R.string.m_tab_discover, Icons.Outlined.Explore),
    Requests(R.string.m_tab_requests, Icons.Outlined.Checklist),
    Library(R.string.m_tab_library, Icons.Outlined.VideoLibrary),
    Profile(R.string.m_tab_profile, Icons.Outlined.AccountCircle),
    Search(R.string.m_tab_search, Icons.Outlined.Search),
}

/** The signed-in app: five tabs in a bottom bar. Only Discover exists so far. */
@Composable
fun MainTabs(account: Account, profile: Profile, userFolderChoice: Boolean, onSignOut: () -> Unit) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val model: MainViewModel = viewModel(key = account.id, factory = viewModelFactory {
        initializer { MainViewModel(TipsarrApi(account.serverUrl, account.token), profile.isAdmin, userFolderChoice) }
    })
    val source = remember(account.id) { ImageSource(account.serverUrl, account.token) }
    val loader = remember(account.id) { createImageLoader(context, source) }
    var tab by rememberSaveable { mutableStateOf(AppTab.Discover) }
    val toasts = remember { ToastCenter(scope) }
    val openTitle: (MediaItem) -> Unit = { model.open(tab, MediaRoute(it.type, it.tmdbId, it.title)) }
    BackHandler(enabled = model.top(tab) != null) { model.back(tab) }

    CompositionLocalProvider(LocalImageSource provides source, LocalAppImageLoader provides loader, LocalToast provides toasts) {
      Box(Modifier.fillMaxSize()) {
        Scaffold(
            containerColor = Tokens.palette.bg,
            bottomBar = {
                NavigationBar(containerColor = Tokens.palette.card) {
                    AppTab.entries.forEach { item ->
                        NavigationBarItem(
                            selected = tab == item,
                            onClick = { tab = item },
                            icon = { Icon(item.icon, contentDescription = null) },
                            label = { Text(stringResource(item.title)) },
                            colors = NavigationBarItemDefaults.colors(
                                selectedIconColor = Tokens.palette.primaryFg,
                                selectedTextColor = Tokens.palette.fg,
                                indicatorColor = Tokens.palette.primary,
                                unselectedIconColor = Tokens.palette.mutedFg,
                                unselectedTextColor = Tokens.palette.mutedFg,
                            ),
                        )
                    }
                }
            },
        ) { padding ->
            Box(Modifier.fillMaxSize().background(Tokens.palette.bg).padding(padding)) {
                val detail = model.top(tab)
                if (detail != null) {
                    key(detail) {
                        MediaDetailScreen(
                            detail, account.serverUrl,
                            onBack = { model.back(tab) },
                            onOpen = { model.open(tab, it) },
                        )
                    }
                } else when (tab) {
                    AppTab.Discover -> DiscoverScreen(model.discover, onOpenSearch = { tab = AppTab.Search }, onOpenItem = openTitle)
                    AppTab.Profile -> ProfileStub(profile.name, onSignOut)
                    else -> StateView(tab.icon, stringResource(tab.title))
                }
            }
        }
        ToastHost(toasts)
      }
    }
}

/** Until the Profile tab exists (Android step 6) it only offers sign out, so testing does not need a reinstall. */
@Composable
private fun ProfileStub(name: String, onSignOut: () -> Unit) {
    androidx.compose.foundation.layout.Column(
        Modifier.fillMaxSize().padding(Tokens.Spacing.x2xl),
        verticalArrangement = androidx.compose.foundation.layout.Arrangement.spacedBy(Tokens.Spacing.lg, androidx.compose.ui.Alignment.CenterVertically),
    ) {
        Text(name, fontSize = Tokens.FontSize.largeTitle, color = Tokens.palette.fg)
        com.brebond.tipsarr.ui.TipsarrButton(stringResource(R.string.m_settings_sign_out), onSignOut, kind = com.brebond.tipsarr.ui.ButtonKind.Secondary)
    }
}
