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
    let account: Account
    let profile: Profile
    @State private var selection: AppTab = .discover
    @State private var toast = ToastCenter()
    @State private var pendingCount = 0
    @State private var ratings: RatingProvider

    init(account: Account, profile: Profile) {
        self.account = account
        self.profile = profile
        _ratings = State(initialValue: RatingProvider(
            api: TipsarrAPI(serverURL: account.serverURL, token: account.token),
            source: RatingSource(rawValue: profile.ratingSource) ?? .tmdb
        ))
    }

    var body: some View {
        TabView(selection: $selection) {
            Tab("m.tab.discover", systemImage: "safari", value: AppTab.discover) {
                DiscoverView(api: api) { selection = .search }
            }
            Tab("m.tab.search", systemImage: "magnifyingglass", value: AppTab.search, role: .search) {
                SearchView(api: api)
            }
            Tab("m.tab.requests", systemImage: "checklist", value: AppTab.requests) {
                RequestsView(api: api, isAdmin: profile.isAdmin) { selection = .discover }
            }
            .badge(profile.isAdmin ? pendingCount : 0)
            Tab("m.tab.library", systemImage: "books.vertical", value: AppTab.library) {
                LibraryView(api: api)
            }
            Tab("m.tab.profile", systemImage: "person.crop.circle", value: AppTab.profile) {
                ProfileView(account: account, profile: profile, api: api) { selection = .requests }
            }
        }
        .environment(\.imageSource, ImageSource(serverURL: account.serverURL, token: account.token))
        .environment(\.appContext, AppContext(api: api, profile: profile, userFolderChoice: session.serverStatus?.userFolderChoice ?? false))
        .environment(\.ratingProvider, ratings)
        .environment(toast)
        .modifier(ToastOverlay(center: toast))
        .onChange(of: profile.ratingSource) { _, new in ratings.setSource(RatingSource(rawValue: new) ?? .tmdb) }
        .task(id: selection) { await refreshPending() }
        .onChange(of: scenePhase) { _, phase in
            if phase == .active { Task { await refreshPending() } }
        }
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

    private var api: TipsarrAPI { TipsarrAPI(serverURL: account.serverURL, token: account.token) }

    private func refreshPending() async {
        guard profile.isAdmin, let counts = try? await api.requestCounts() else { return }
        pendingCount = counts.pending
    }
}
