import SwiftUI
import UserNotifications

enum AppTab: Hashable {
    case teasers, discover, search, requests, library, profile
}

/// iOS 26 floating tab bar.
struct MainTabView: View {
    @Environment(SessionManager.self) private var session
    @Environment(AppSettings.self) private var settings
    @Environment(\.scenePhase) private var scenePhase
    @Environment(\.horizontalSizeClass) private var sizeClass
    let account: Account
    let profile: Profile
    @State private var router = AppRouter()
    @State private var toast = ToastCenter()
    @State private var pendingCount = 0
    @State private var ratings: RatingProvider
    /// The Teasers feed and its first trailers are prepared here, so opening the tab shows a picture at once.
    @State private var teasersModel: TeasersModel
    @State private var teaserVideos: TeaserVideoPool
    @State private var live = LiveUpdates()
    @State private var pane = DetailPane()
    @State private var foldedHinge = false
    /// The Teasers tab only exists while the server reports `features.teasers`.
    private var teasersOn: Bool { session.serverStatus?.teasersAvailable ?? false }
    private var tabList: [AppTab] { AppTab.tabs(teasers: teasersOn) }
    /// Tabs opened at least once; wide windows keep them alive so each tab keeps its navigation and scroll position.
    @State private var visited: Set<AppTab> = [.discover]

    init(account: Account, profile: Profile) {
        self.account = account
        self.profile = profile
        _ratings = State(initialValue: RatingProvider(
            api: TipsarrAPI(serverURL: account.serverURL, token: account.token),
            source: RatingSource(rawValue: profile.ratingSource) ?? .tmdb
        ))
        _teasersModel = State(initialValue: TeasersModel(api: TipsarrAPI(serverURL: account.serverURL, token: account.token)))
        _teaserVideos = State(initialValue: TeaserVideoPool(token: account.token))
    }

    var body: some View {
        GeometryReader { proxy in
            shell(ShellStyle.style(size: proxy.size, compact: sizeClass == .compact, hinge: FoldInfo.verticalHinge(proxy)), trailingInset: proxy.safeAreaInsets.trailing)
        }
        .onGeometryChange(for: Bool.self, of: { FoldInfo.verticalHinge($0) != nil }) { foldedHinge = $0 }
        // Attached before the environment values below: a sheet only inherits what is injected inside it.
        .sheet(isPresented: $router.profileOpen) {
            ProfileView(account: account, profile: profile, api: api, path: $router.profilePath) {
                router.profileOpen = false
                router.tab = .requests
            }
            .tipsarrSheet(detents: [.large])
        }
        .sheet(isPresented: $router.settingsOpen) {
            SettingsScreen(account: account, profile: profile, path: $router.settingsPath) { router.settingsOpen = false }
                .tipsarrSheet(detents: [.large], sizing: .page)
        }
        .environment(\.imageSource, ImageSource(serverURL: account.serverURL, token: account.token))
        .environment(\.appContext, AppContext(api: api, profile: profile))
        .environment(\.ratingProvider, ratings)
        .environment(\.liveUpdates, live)
        .environment(toast)
        .modifier(ToastOverlay(center: toast))
        // Hardware keyboard: Cmd+1...5 switch tabs, Cmd+F opens Search.
        .background {
            ForEach(Array(tabList.enumerated()), id: \.element) { index, tab in
                Button("") { router.tab = tab }
                    .keyboardShortcut(KeyEquivalent(Character(String(index + 1))), modifiers: .command)
                    .accessibilityHidden(true)
            }
            Button("") { router.tab = .search }
                .keyboardShortcut("f", modifiers: .command)
                .accessibilityHidden(true)
        }
        // A server without trailer files has no Teasers tab: if it was selected, Discover takes over.
        .onChange(of: teasersOn, initial: true) { _, on in
            if !on && router.tab == .teasers { router.tab = .discover }
        }
        .task { await session.refreshStatus() }
        .task(id: teasersOn) {
            // The first two trailers start buffering now (paused), long before the tab is opened.
            guard teasersOn else { return }
            await teasersModel.loadIfNeeded()
            for teaser in teasersModel.items.prefix(2) { _ = teaserVideos.video(for: teaser) }
        }
        .onChange(of: profile.ratingSource) { _, new in ratings.setSource(RatingSource(rawValue: new) ?? .tmdb) }
        .task(id: router.tab) { await refreshPending() }
        .onChange(of: scenePhase) { _, phase in
            if phase == .active { Task { await refreshPending(); await session.refreshStatus() } }
        }
        // Server stream: only while the app is in the foreground.
        .task(id: scenePhase) {
            if scenePhase == .active { live.start(server: account.serverURL, token: account.token) } else { live.stop() }
        }
        .onChange(of: live.requestsTick) { Task { await refreshPending() } }
        .onChange(of: session.pendingLink) { Task { await consumeLink() } }
        .task { await consumeLink() }
        .onDisappear { live.stop() }
        .onChange(of: pendingCount) { _, count in
            // Same number on the app icon as on the Requests tab (admins only).
            Task { try? await UNUserNotificationCenter.current().setBadgeCount(count) }
        }
        .task {
            // The server knows the language the app shows (titles and descriptions follow it).
            let code = Bundle.main.preferredLocalizations.first ?? "en"
            if !profile.language.lowercased().hasPrefix(code), !(profile.language.isEmpty && code == "en"),
               let updated = try? await api.updatePreferences(language: code) {
                session.update(profile: updated)
            }
        }
        .task {
            // No region yet: the iPhone's one, which the person can change in Settings.
            if profile.region.isEmpty, let code = Locale.current.region?.identifier, code.count == 2,
               let updated = try? await api.updatePreferences(region: code.uppercased()) {
                session.update(profile: updated)
            }
        }
        .task {
            // People who manage requests see their pending count on the icon, so they are asked at once (alerts, sounds and the badge together).
            if profile.can(.manageRequests) { _ = try? await UNUserNotificationCenter.current().requestAuthorization(options: [.alert, .sound, .badge]) }
            await PushManager.shared.attach(account: account, pushAvailable: session.serverStatus?.pushAvailable ?? false)
        }
        // A new language or account rebuilds the tabs, so every list is fetched again in that language.
        .id("\(account.id)|\(settings.language ?? "")|\(profile.region)")
    }

    /// The four tabs inside the system bar (iPhone, narrow windows), or with our own bar: a sidebar on a
    /// landscape iPad window, a floating bar at the bottom of a portrait one.
    @ViewBuilder private func shell(_ style: ShellStyle, trailingInset: CGFloat = 0) -> some View {
        let badges: [AppTab: Int] = profile.can(.manageRequests) ? [.requests: pendingCount] : [:]
        if case .dualPane(let hinge) = style {
            // Left of the hinge: the tab content with its own bar. Right of it: the opened title or request.
            HStack(spacing: 0) {
                tabs(style)
                    // Navigation stacks take the window's trailing safe area (clock strip) although this pane ends at the hinge.
                    .environment(\.paneTrailingInset, trailingInset)
                    .environment(\.detailPane, pane)
                    .safeAreaInset(edge: .bottom, spacing: 0) { Color.clear.frame(height: 88) }
                    .overlay(alignment: .bottom) {
                        FloatingTabBar(selection: Binding(get: { router.tab }, set: { router.tab = $0; pane.content = nil }), tabs: tabList, badges: badges, showsTitles: false).padding(.bottom, Tokens.Spacing.sm)
                    }
                    .frame(width: max(hinge.leftWidth, 0))
                Color.clear.frame(width: max(hinge.rightStart - hinge.leftWidth, 0))
                DetailPaneView(pane: pane, tab: router.tab, account: account, profile: profile, trailingInset: trailingInset)
                    .frame(maxWidth: .infinity)
            }
        } else {
            HStack(spacing: 0) {
                if style == .sidebar {
                    AppSidebar(selection: $router.tab, tabs: tabList, account: account, profile: profile, badges: badges) { router.openProfile() }
                }
                tabs(style)
                    .safeAreaInset(edge: .bottom, spacing: 0) {
                        if style == .bottomBar { Color.clear.frame(height: 96) }
                    }
                    .overlay(alignment: .bottom) {
                        if style == .bottomBar { FloatingTabBar(selection: $router.tab, tabs: tabList, badges: badges).padding(.bottom, Tokens.Spacing.sm) }
                    }
            }
        }
    }

    @ViewBuilder private func tabs(_ style: ShellStyle) -> some View {
        if style == .system {
            // iPhone and narrow windows: the system tab bar.
            TabView(selection: $router.tab) {
                if teasersOn {
                    Tab("m.tab.teasers", systemImage: "play.rectangle", value: AppTab.teasers) { content(.teasers) }
                }
                Tab("m.tab.discover", systemImage: "safari", value: AppTab.discover) { content(.discover) }
                Tab("m.tab.requests", systemImage: "checklist", value: AppTab.requests) { content(.requests) }
                    .badge(profile.can(.manageRequests) ? pendingCount : 0)
                Tab("m.tab.library", systemImage: "books.vertical", value: AppTab.library) { content(.library) }
                // A normal last tab, not the separate search tab: its field stays visible under the title.
                Tab("m.tab.search", systemImage: "magnifyingglass", value: AppTab.search) { content(.search) }
            }
        } else {
            // Wide windows draw their own bar (sidebar or floating bar), so no TabView: the system bar cannot be hidden on iPad.
            ZStack {
                ForEach(tabList, id: \.self) { tab in
                    if visited.contains(tab) {
                        content(tab, bottomInset: style == .bottomBar ? 96 : (style == .sidebar ? 0 : 88))
                            .opacity(router.tab == tab ? 1 : 0)
                            .allowsHitTesting(router.tab == tab)
                            .accessibilityHidden(router.tab != tab)
                    }
                }
            }
            .onChange(of: router.tab, initial: true) { _, tab in visited.insert(tab) }
        }
    }

    @ViewBuilder private func content(_ tab: AppTab, bottomInset: CGFloat = 0) -> some View {
        switch tab {
        case .teasers:
            TeasersView(model: teasersModel, pool: teaserVideos, isSelected: router.tab == .teasers, path: $router.teasersPath, bottomInset: bottomInset) {
                // The server stopped serving Teasers while the tab was open.
                Task { await session.refreshStatus() }
                router.tab = .discover
            }
        case .discover: DiscoverView(api: api, path: $router.discoverPath, openProfile: { router.openProfile() }, openSettings: { router.openSettings() })
        case .search: SearchView(api: api)
        case .requests: RequestsView(api: api, canManage: profile.can(.manageRequests), canViewAll: profile.can(.requestView), path: $router.requestsPath, selection: $router.requestsSelection) { router.tab = .discover }
        case .library: LibraryView(api: api, path: $router.libraryPath)
        case .profile: EmptyView() // not a tab: Profile is the sheet above
        }
    }

    /// True while the Duo is unfolded (the shell injected `pane`).
    private var dualPaneActive: Bool { foldedHinge }

    private var api: TipsarrAPI { TipsarrAPI(serverURL: account.serverURL, token: account.token) }

    /// A link for this account's server opens its screen; links for another server wait for that account.
    private func consumeLink() async {
        guard let link = session.pendingLink, link.host == account.serverURL.host()?.lowercased() else { return }
        session.clearPendingLink()
        await router.open(link.target, api: api, pane: dualPaneActive ? pane : nil)
    }

    private func refreshPending() async {
        // Not an admin (anymore): the icon must not keep the count a previous account left there.
        guard profile.can(.manageRequests) else {
            pendingCount = 0
            try? await UNUserNotificationCenter.current().setBadgeCount(0)
            return
        }
        guard let counts = try? await api.requestCounts() else { return }
        pendingCount = counts.pending
        // Set here too: onChange does not fire when a rebuilt view fetches the number the icon lacks.
        try? await UNUserNotificationCenter.current().setBadgeCount(counts.pending)
    }
}
