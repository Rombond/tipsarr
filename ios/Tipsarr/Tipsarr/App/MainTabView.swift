import SwiftUI
import UserNotifications

enum AppTab: Hashable {
    case discover, search, requests, library, profile
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
    @State private var live = LiveUpdates()
    @State private var pane = DetailPane()
    @State private var foldedHinge = false
    /// Tabs opened at least once; wide windows keep them alive so each tab keeps its navigation and scroll position.
    @State private var visited: Set<AppTab> = [.discover]

    init(account: Account, profile: Profile) {
        self.account = account
        self.profile = profile
        _ratings = State(initialValue: RatingProvider(
            api: TipsarrAPI(serverURL: account.serverURL, token: account.token),
            source: RatingSource(rawValue: profile.ratingSource) ?? .tmdb
        ))
    }

    var body: some View {
        GeometryReader { proxy in
            shell(ShellStyle.style(size: proxy.size, compact: sizeClass == .compact, hinge: FoldInfo.verticalHinge(proxy)), trailingInset: proxy.safeAreaInsets.trailing)
        }
        .onGeometryChange(for: Bool.self, of: { FoldInfo.verticalHinge($0) != nil }) { foldedHinge = $0 }
        .environment(\.imageSource, ImageSource(serverURL: account.serverURL, token: account.token))
        .environment(\.appContext, AppContext(api: api, profile: profile, userFolderChoice: session.serverStatus?.userFolderChoice ?? false))
        .environment(\.ratingProvider, ratings)
        .environment(\.liveUpdates, live)
        .environment(toast)
        .modifier(ToastOverlay(center: toast))
        // Hardware keyboard: Cmd+1...5 switch tabs, Cmd+F opens Search.
        .background {
            ForEach(Array(AppTab.ordered.enumerated()), id: \.element) { index, tab in
                Button("") { router.tab = tab }
                    .keyboardShortcut(KeyEquivalent(Character(String(index + 1))), modifiers: .command)
                    .accessibilityHidden(true)
            }
            Button("") { router.tab = .search }
                .keyboardShortcut("f", modifiers: .command)
                .accessibilityHidden(true)
        }
        .onChange(of: profile.ratingSource) { _, new in ratings.setSource(RatingSource(rawValue: new) ?? .tmdb) }
        .task(id: router.tab) { await refreshPending() }
        .onChange(of: scenePhase) { _, phase in
            if phase == .active { Task { await refreshPending() } }
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
            // Admins see their pending count on the icon, so they are asked at once (alerts, sounds and the badge together).
            if profile.isAdmin { _ = try? await UNUserNotificationCenter.current().requestAuthorization(options: [.alert, .sound, .badge]) }
            await PushManager.shared.attach(account: account, pushAvailable: session.serverStatus?.pushAvailable ?? false)
        }
        // A new language or account rebuilds the tabs, so every list is fetched again in that language.
        .id("\(account.id)|\(settings.language ?? "")|\(profile.region)")
    }

    /// The five tabs inside the system bar (iPhone, narrow windows), or with our own bar: a sidebar on a
    /// landscape iPad window, a floating bar at the bottom of a portrait one.
    @ViewBuilder private func shell(_ style: ShellStyle, trailingInset: CGFloat = 0) -> some View {
        let badges: [AppTab: Int] = profile.isAdmin ? [.requests: pendingCount] : [:]
        if case .dualPane(let hinge) = style {
            // Left of the hinge: the tab content with its own bar. Right of it: the opened title or request.
            HStack(spacing: 0) {
                tabs(style)
                    // Navigation stacks take the window's trailing safe area (clock strip) although this pane ends at the hinge.
                    .environment(\.paneTrailingInset, trailingInset)
                    .environment(\.detailPane, pane)
                    .safeAreaInset(edge: .bottom, spacing: 0) { Color.clear.frame(height: 88) }
                    .overlay(alignment: .bottom) {
                        FloatingTabBar(selection: Binding(get: { router.tab }, set: { router.tab = $0; pane.content = nil }), badges: badges, showsTitles: false).padding(.bottom, Tokens.Spacing.sm)
                    }
                    .frame(width: max(hinge.leftWidth, 0))
                Color.clear.frame(width: max(hinge.rightStart - hinge.leftWidth, 0))
                DetailPaneView(pane: pane, tab: router.tab, account: account, profile: profile, trailingInset: trailingInset)
                    .frame(maxWidth: .infinity)
            }
        } else {
            HStack(spacing: 0) {
                if style == .sidebar {
                    AppSidebar(selection: $router.tab, account: account, profile: profile, badges: badges)
                }
                tabs(style)
                    .safeAreaInset(edge: .bottom, spacing: 0) {
                        if style == .bottomBar { Color.clear.frame(height: 96) }
                    }
                    .overlay(alignment: .bottom) {
                        if style == .bottomBar { FloatingTabBar(selection: $router.tab, badges: badges).padding(.bottom, Tokens.Spacing.sm) }
                    }
            }
        }
    }

    @ViewBuilder private func tabs(_ style: ShellStyle) -> some View {
        if style == .system {
            // iPhone and narrow windows: the system tab bar.
            TabView(selection: $router.tab) {
                Tab("m.tab.discover", systemImage: "safari", value: AppTab.discover) { content(.discover) }
                Tab("m.tab.requests", systemImage: "checklist", value: AppTab.requests) { content(.requests) }
                    .badge(profile.isAdmin ? pendingCount : 0)
                Tab("m.tab.library", systemImage: "books.vertical", value: AppTab.library) { content(.library) }
                Tab("m.tab.profile", systemImage: "person.crop.circle", value: AppTab.profile) { content(.profile) }
                // A normal last tab, not the separate search tab: its field stays visible under the title.
                Tab("m.tab.search", systemImage: "magnifyingglass", value: AppTab.search) { content(.search) }
            }
        } else {
            // Wide windows draw their own bar (sidebar or floating bar), so no TabView: the system bar cannot be hidden on iPad.
            ZStack {
                ForEach(AppTab.ordered, id: \.self) { tab in
                    if visited.contains(tab) {
                        content(tab)
                            .opacity(router.tab == tab ? 1 : 0)
                            .allowsHitTesting(router.tab == tab)
                            .accessibilityHidden(router.tab != tab)
                    }
                }
            }
            .onChange(of: router.tab, initial: true) { _, tab in visited.insert(tab) }
        }
    }

    @ViewBuilder private func content(_ tab: AppTab) -> some View {
        switch tab {
        case .discover: DiscoverView(api: api, path: $router.discoverPath)
        case .search: SearchView(api: api)
        case .requests: RequestsView(api: api, isAdmin: profile.isAdmin, path: $router.requestsPath, selection: $router.requestsSelection) { router.tab = .discover }
        case .library: LibraryView(api: api, path: $router.libraryPath)
        case .profile: ProfileView(account: account, profile: profile, api: api, path: $router.profilePath) { router.tab = .requests }
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
        guard profile.isAdmin, let counts = try? await api.requestCounts() else { return }
        pendingCount = counts.pending
    }
}
