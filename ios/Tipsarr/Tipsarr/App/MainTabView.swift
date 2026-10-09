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
            shell(ShellStyle.style(size: proxy.size, compact: sizeClass == .compact))
        }
        .environment(\.imageSource, ImageSource(serverURL: account.serverURL, token: account.token))
        .environment(\.appContext, AppContext(api: api, profile: profile, userFolderChoice: session.serverStatus?.userFolderChoice ?? false))
        .environment(\.ratingProvider, ratings)
        .environment(\.liveUpdates, live)
        .environment(toast)
        .modifier(ToastOverlay(center: toast))
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
            if profile.isAdmin { _ = try? await UNUserNotificationCenter.current().requestAuthorization(options: [.badge]) }
        }
        // A new language or account rebuilds the tabs, so every list is fetched again in that language.
        .id("\(account.id)|\(settings.language ?? "")")
    }

    /// The five tabs inside the system bar (iPhone, narrow windows), or with our own bar: a sidebar on a
    /// landscape iPad window, a floating bar at the bottom of a portrait one.
    @ViewBuilder private func shell(_ style: ShellStyle) -> some View {
        let badges: [AppTab: Int] = profile.isAdmin ? [.requests: pendingCount] : [:]
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

    private func tabs(_ style: ShellStyle) -> some View {
        TabView(selection: $router.tab) {
            Tab("m.tab.discover", systemImage: "safari", value: AppTab.discover) {
                DiscoverView(api: api, path: $router.discoverPath) { router.tab = .search }
            }
            // The separate search tab (field at the top right of the bar) is for the system bar only; wide windows put the field in the page.
            Tab("m.tab.search", systemImage: "magnifyingglass", value: AppTab.search, role: style == .system && sizeClass == .compact ? .search : nil) {
                SearchView(api: api)
            }
            Tab("m.tab.requests", systemImage: "checklist", value: AppTab.requests) {
                RequestsView(api: api, isAdmin: profile.isAdmin, path: $router.requestsPath) { router.tab = .discover }
            }
            .badge(profile.isAdmin ? pendingCount : 0)
            Tab("m.tab.library", systemImage: "books.vertical", value: AppTab.library) {
                LibraryView(api: api, path: $router.libraryPath)
            }
            Tab("m.tab.profile", systemImage: "person.crop.circle", value: AppTab.profile) {
                ProfileView(account: account, profile: profile, api: api) { router.tab = .requests }
            }
        }
        .toolbarVisibility(style == .system ? .automatic : .hidden, for: .tabBar)
    }

    private var api: TipsarrAPI { TipsarrAPI(serverURL: account.serverURL, token: account.token) }

    /// A link for this account's server opens its screen; links for another server wait for that account.
    private func consumeLink() async {
        guard let link = session.pendingLink, link.host == account.serverURL.host()?.lowercased() else { return }
        session.clearPendingLink()
        await router.open(link.target, api: api)
    }

    private func refreshPending() async {
        guard profile.isAdmin, let counts = try? await api.requestCounts() else { return }
        pendingCount = counts.pending
    }
}
