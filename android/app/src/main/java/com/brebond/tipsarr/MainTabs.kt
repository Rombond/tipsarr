package com.brebond.tipsarr

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.width
import androidx.compose.material.icons.outlined.ViewSidebar
import androidx.compose.material3.NavigationRail
import androidx.compose.material3.NavigationRailItem
import androidx.compose.material3.NavigationRailItemDefaults
import androidx.compose.material3.VerticalDivider
import androidx.compose.ui.unit.dp
import com.brebond.tipsarr.core.support.rememberVerticalHinge
import com.brebond.tipsarr.features.discover.TrendingPane
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.WindowInsetsSides
import androidx.compose.foundation.layout.only
import androidx.compose.foundation.layout.statusBars
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.layout.systemBars
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.AccountCircle
import androidx.compose.material.icons.outlined.Checklist
import androidx.compose.material.icons.outlined.Explore
import androidx.compose.material.icons.outlined.Search
import androidx.compose.material.icons.outlined.VideoLibrary
import androidx.compose.material3.Badge
import androidx.compose.material3.BadgedBox
import androidx.compose.material3.Icon
import androidx.compose.material3.NavigationBar
import androidx.compose.material3.NavigationBarItem
import androidx.compose.material3.NavigationBarItemDefaults
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import com.brebond.tipsarr.core.api.request
import com.brebond.tipsarr.core.live.LocalLive
import com.brebond.tipsarr.core.live.OnTick
import com.brebond.tipsarr.core.navigation.DeepLinkTarget
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
import com.brebond.tipsarr.features.admin.IssueDetailScreen
import com.brebond.tipsarr.features.admin.IssuesScreen
import com.brebond.tipsarr.features.admin.PosterListScreen
import com.brebond.tipsarr.features.admin.StatsScreen
import com.brebond.tipsarr.features.admin.SyncScreen
import com.brebond.tipsarr.features.admin.UserDetailScreen
import com.brebond.tipsarr.features.admin.UsersScreen
import com.brebond.tipsarr.features.library.LibraryScreen
import com.brebond.tipsarr.features.person.PersonScreen
import com.brebond.tipsarr.features.requests.RequestDetailScreen
import com.brebond.tipsarr.features.requests.RequestsScreen
import com.brebond.tipsarr.features.search.GenreScreen
import com.brebond.tipsarr.features.search.SearchScreen
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
import com.brebond.tipsarr.core.api.RatingSource
import com.brebond.tipsarr.core.auth.SessionManager
import com.brebond.tipsarr.core.support.AppSettings
import com.brebond.tipsarr.core.support.LocalRatings
import com.brebond.tipsarr.features.profile.PictureSheet
import com.brebond.tipsarr.features.profile.ProfileScreen
import com.brebond.tipsarr.features.settings.AccountsScreen
import com.brebond.tipsarr.features.settings.DevicesScreen
import com.brebond.tipsarr.features.settings.HiddenScreen
import com.brebond.tipsarr.features.settings.SettingsScreen
import com.brebond.tipsarr.features.settings.WatchlistScreen
import kotlinx.coroutines.launch
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
fun MainTabs(account: Account, profile: Profile, userFolderChoice: Boolean, serverVersion: String?, session: SessionManager, settings: AppSettings, onLanguageChanged: () -> Unit) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val model: MainViewModel = viewModel(key = account.id, factory = viewModelFactory {
        initializer { MainViewModel(TipsarrApi(account.serverUrl, account.token), context.applicationContext, profile.isAdmin, userFolderChoice, RatingSource.from(profile.ratingSource)) }
    })
    val source = remember(account.id) { ImageSource(account.serverUrl, account.token) }
    val loader = remember(account.id) { createImageLoader(context, source) }
    var tab by rememberSaveable { mutableStateOf(AppTab.Discover) }
    val toasts = remember { ToastCenter(scope) }
    val openTitle: (MediaItem) -> Unit = { model.openTitle(tab, MediaRoute(it.type, it.tmdbId, it.title)) }
    LaunchedEffect(tab) { model.refreshPending() }
    LaunchedEffect(profile.ratingSource) { model.ratings.changeSource(RatingSource.from(profile.ratingSource)) }
    var showPicture by remember { mutableStateOf(false) }

    // The event stream runs only while the app is in the foreground.
    val lifecycle = androidx.lifecycle.compose.LocalLifecycleOwner.current.lifecycle
    DisposableEffect(lifecycle, account.id) {
        val observer = androidx.lifecycle.LifecycleEventObserver { _, event ->
            when (event) {
                androidx.lifecycle.Lifecycle.Event.ON_START -> model.live.start(account.serverUrl, account.token)
                androidx.lifecycle.Lifecycle.Event.ON_STOP -> model.live.stop()
                else -> {}
            }
        }
        lifecycle.addObserver(observer)
        if (lifecycle.currentState.isAtLeast(androidx.lifecycle.Lifecycle.State.STARTED)) model.live.start(account.serverUrl, account.token)
        onDispose { lifecycle.removeObserver(observer); model.live.stop() }
    }
    OnTick(model.live.requestsTick) { model.refreshPending() }

    // A link for this account's server opens its screen; links for another server wait for that account.
    val link by session.pendingLink.collectAsState()
    LaunchedEffect(link) {
        val pending = link ?: return@LaunchedEffect
        if (pending.host != account.host) return@LaunchedEffect
        session.clearPendingLink()
        when (val target = pending.target) {
            is DeepLinkTarget.Media -> { model.reset(AppTab.Discover); model.openTitle(AppTab.Discover, MediaRoute(target.type, target.tmdbId, "")); tab = AppTab.Discover }
            is DeepLinkTarget.Request -> {
                val record = runCatching { model.api.request(target.id) }.getOrNull()
                model.reset(AppTab.Requests)
                if (record != null) model.openRequest(AppTab.Requests, record)
                tab = AppTab.Requests
            }
            is DeepLinkTarget.Issue -> { model.reset(AppTab.Profile); model.open(AppTab.Profile, Screen.Issue(target.id)); tab = AppTab.Profile }
            DeepLinkTarget.Discover -> { model.reset(AppTab.Discover); tab = AppTab.Discover }
            DeepLinkTarget.Library -> { model.reset(AppTab.Library); tab = AppTab.Library }
            DeepLinkTarget.Requests -> { model.reset(AppTab.Requests); tab = AppTab.Requests }
        }
    }
    val signOut: () -> Unit = { scope.launch { session.signOut(account) } }
    BackHandler(enabled = model.top(tab) != null) { model.back(tab) }

    val hinge = rememberVerticalHinge()

    // How the tabs are shown: a bottom bar on a phone, a rail on a wide window, and on a folded-open screen two panes
    // either side of the hinge (a list on the left, what was opened on the right).
    val renderScreen: @Composable (Screen) -> Unit = { top ->
        key(top) {
            when (top) {
                            is Screen.Title -> MediaDetailScreen(
                                top.model, account.serverUrl,
                                onBack = { model.back(tab) },
                                onOpen = { model.openTitle(tab, it) },
                                onOpenPerson = { model.openPerson(tab, it) },
                            )
                            is Screen.Person -> PersonScreen(top.model, onBack = { model.back(tab) }, onOpenItem = openTitle)
                            is Screen.Genre -> GenreScreen(top.route, top.list, onBack = { model.back(tab) }, onOpenItem = openTitle)
                            is Screen.Request -> RequestDetailScreen(
                                top.record, model.requests, profile.name,
                                onBack = { model.back(tab) },
                                onOpenTitle = { model.openTitle(tab, it) },
                            )
                            Screen.Settings -> SettingsScreen(
                                account, profile, model.api, settings, session.accounts.accounts.size, serverVersion,
                                onBack = { model.back(tab) },
                                onProfileChanged = { session.update(it) },
                                onLanguageChanged = onLanguageChanged,
                                onOpenAccounts = { model.open(tab, Screen.Accounts) },
                                onOpenDevices = { model.open(tab, Screen.Devices) },
                                onOpenHidden = { model.open(tab, Screen.Hidden) },
                                onChangePicture = { showPicture = true },
                                onSignOut = signOut,
                            )
                            Screen.Accounts -> AccountsScreen(
                                session.accounts.accounts, session.accounts.activeId, model.avatarVersion,
                                onBack = { model.back(tab) },
                                onSwitch = { scope.launch { session.switchTo(it) } },
                                onAdd = { session.beginAddAccount() },
                                onSignOut = { scope.launch { session.signOut(it) } },
                            )
                            Screen.Devices -> DevicesScreen(model.api, onBack = { model.back(tab) })
                            Screen.Watchlist -> WatchlistScreen(model.api, onBack = { model.back(tab) }, onOpen = openTitle)
                            Screen.Hidden -> HiddenScreen(model.api, onBack = { model.back(tab) }, onOpen = openTitle)
                            Screen.Issues -> IssuesScreen(model.api, onBack = { model.back(tab) }, onOpen = { model.open(tab, Screen.Issue(it)) })
                            is Screen.Issue -> IssueDetailScreen(model.api, top.id, profile.isAdmin, onBack = { model.back(tab) }, onOpenTitle = { model.openTitle(tab, it) })
                            Screen.Users -> UsersScreen(model.api, onBack = { model.back(tab) }, onOpen = { model.open(tab, Screen.User(it)) })
                            is Screen.User -> UserDetailScreen(model.api, top.id, profile.id, onBack = { model.back(tab) })
                            Screen.Sync -> SyncScreen(model.api, onBack = { model.back(tab) })
                            Screen.Stats -> StatsScreen(model.api, profile.isAdmin, profile, onBack = { model.back(tab) }, onOpenTitle = { model.openTitle(tab, it) }, onSeeAll = { model.open(tab, Screen.PosterList(it)) })
                            is Screen.PosterList -> PosterListScreen(top.route, onBack = { model.back(tab) }, onOpen = { model.openTitle(tab, it) })
            }
        }
    }
    val renderRoot: @Composable (AppTab) -> Unit = { current ->
        when (current) {

                    AppTab.Discover -> DiscoverScreen(model.discover, onOpenSearch = { tab = AppTab.Search }, onOpenItem = openTitle)
                    AppTab.Library -> LibraryScreen(model.library, onOpen = { model.openTitle(tab, it) })
                    AppTab.Requests -> RequestsScreen(model.requests, onOpen = { model.openRequest(tab, it) }, onOpenDiscover = { tab = AppTab.Discover })
                    AppTab.Search -> SearchScreen(
                        model.search,
                        onOpenItem = openTitle,
                        onOpenPerson = { model.openPerson(tab, it) },
                        onOpenGenre = { type, genre -> model.openGenre(tab, type, genre.id, genre.name) },
                    )
                    AppTab.Profile -> ProfileScreen(
                        model.profile, profile, model.avatarVersion,
                        onOpenSettings = { model.open(tab, Screen.Settings) },
                        onOpenRequests = { tab = AppTab.Requests },
                        onOpenWatchlist = { model.open(tab, Screen.Watchlist) },
                        onOpenTitle = { model.openTitle(tab, it) },
                        onOpenRequest = { model.openRequest(tab, it) },
                        onChangePicture = { showPicture = true },
                        onOpenStats = { model.open(tab, Screen.Stats) },
                        onOpenIssues = { model.open(tab, Screen.Issues) },
                        onOpenUsers = { model.open(tab, Screen.Users) },
                        onOpenSync = { model.open(tab, Screen.Sync) },
                    )
            else -> StateView(current.icon, stringResource(current.title))
        }
    }
    /** What the right pane shows while nothing is opened. */
    val renderEmpty: @Composable (AppTab) -> Unit = { current ->
        when (current) {
            AppTab.Profile -> renderScreen(Screen.Stats)
            AppTab.Search -> TrendingPane(model.trending, openTitle)
            else -> StateView(Icons.Outlined.ViewSidebar, stringResource(R.string.m_pane_select))
        }
    }
    val navigationBar: @Composable () -> Unit = {
        NavigationBar(containerColor = Tokens.palette.card) {
            AppTab.entries.forEach { item ->
                NavigationBarItem(
                    selected = tab == item,
                    onClick = { tab = item },
                    icon = { TabIcon(item, model.pendingCount) },
                    label = { Text(stringResource(item.title)) },
                    colors = NavigationBarItemDefaults.colors(
                        selectedIconColor = Tokens.palette.primaryFg, selectedTextColor = Tokens.palette.fg, indicatorColor = Tokens.palette.primary,
                        unselectedIconColor = Tokens.palette.mutedFg, unselectedTextColor = Tokens.palette.mutedFg,
                    ),
                )
            }
        }
    }

    CompositionLocalProvider(LocalImageSource provides source, LocalAppImageLoader provides loader, LocalToast provides toasts, LocalRatings provides model.ratings, LocalLive provides model.live) {
        Box(Modifier.fillMaxSize().background(Tokens.palette.bg)) {
            BoxWithConstraints(Modifier.fillMaxSize()) {
                val rail = hinge == null && maxWidth >= 840.dp
                val top = model.top(tab)
                // Requests and Search split into list and detail on a wide window; every tab does next to a hinge.
                val split = hinge != null || (maxWidth >= 700.dp && (tab == AppTab.Requests || tab == AppTab.Search))
                val single: @Composable () -> Unit = {
                    if (top != null) renderScreen(top) else Box(Modifier.fillMaxSize().statusBarsPadding()) { renderRoot(tab) }
                }
                val body: @Composable () -> Unit = {
                    if (!split) {
                        single()
                    } else {
                        Row(Modifier.fillMaxSize()) {
                            Box(Modifier.width(400.dp).fillMaxHeight().statusBarsPadding()) { renderRoot(tab) }
                            VerticalDivider(color = Tokens.palette.border)
                            Box(Modifier.weight(1f).fillMaxHeight()) {
                                if (top != null) renderScreen(top) else Box(Modifier.fillMaxSize().statusBarsPadding()) { renderEmpty(tab) }
                            }
                        }
                    }
                }
                when {
                    hinge != null -> Row(Modifier.fillMaxSize()) {
                        Scaffold(
                            modifier = Modifier.width(hinge.left),
                            containerColor = Tokens.palette.bg,
                            contentWindowInsets = WindowInsets.systemBars.only(WindowInsetsSides.Start),
                            bottomBar = navigationBar,
                        ) { padding ->
                            Box(Modifier.fillMaxSize().padding(padding).statusBarsPadding()) { renderRoot(tab) }
                        }
                        Spacer(Modifier.width(hinge.right - hinge.left).fillMaxHeight())
                        Box(Modifier.weight(1f).fillMaxHeight()) {
                            if (top != null) renderScreen(top) else Box(Modifier.fillMaxSize().statusBarsPadding()) { renderEmpty(tab) }
                        }
                    }
                    rail -> Row(Modifier.fillMaxSize()) {
                        NavigationRail(containerColor = Tokens.palette.card, modifier = Modifier.statusBarsPadding()) {
                            Spacer(Modifier.weight(1f))
                            AppTab.entries.forEach { item ->
                                NavigationRailItem(
                                    selected = tab == item,
                                    onClick = { tab = item },
                                    icon = { TabIcon(item, model.pendingCount) },
                                    label = { Text(stringResource(item.title)) },
                                    colors = NavigationRailItemDefaults.colors(
                                        selectedIconColor = Tokens.palette.primaryFg, selectedTextColor = Tokens.palette.fg, indicatorColor = Tokens.palette.primary,
                                        unselectedIconColor = Tokens.palette.mutedFg, unselectedTextColor = Tokens.palette.mutedFg,
                                    ),
                                )
                            }
                            Spacer(Modifier.weight(1f))
                        }
                        Box(Modifier.weight(1f).fillMaxHeight()) { body() }
                    }
                    else -> Scaffold(
                        containerColor = Tokens.palette.bg,
                        // Top insets belong to each screen: tab roots pad for the status bar, pushed screens draw under it.
                        contentWindowInsets = WindowInsets.systemBars.only(WindowInsetsSides.Horizontal),
                        bottomBar = navigationBar,
                    ) { padding -> Box(Modifier.fillMaxSize().padding(padding)) { body() } }
                }
            }
            ToastHost(toasts)
            if (showPicture) PictureSheet(model.api, onDismiss = { showPicture = false }, onChanged = model::avatarChanged)
        }
    }
}

@Composable
private fun TabIcon(item: AppTab, pending: Int) {
    BadgedBox(badge = { if (item == AppTab.Requests && pending > 0) Badge { Text(pending.toString()) } }) {
        Icon(item.icon, contentDescription = null)
    }
}
