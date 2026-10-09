import SwiftUI

enum AppTab: Hashable {
    case discover, search, requests, library, profile
}

/// iOS 26 floating tab bar. Search, Requests, Library and Profile are filled in by later steps.
struct MainTabView: View {
    @Environment(SessionManager.self) private var session
    let account: Account
    let profile: Profile
    @State private var selection: AppTab = .discover
    @State private var toast = ToastCenter()
    @State private var pendingCount = 0

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
                ProfileView(account: account, profile: profile)
            }
        }
        .environment(\.imageSource, ImageSource(serverURL: account.serverURL, token: account.token))
        .environment(\.appContext, AppContext(api: api, profile: profile, userFolderChoice: session.serverStatus?.userFolderChoice ?? false))
        .environment(toast)
        .modifier(ToastOverlay(center: toast))
        .task(id: selection) {
            // Keeps the admin's pending badge fresh whenever the tab changes.
            if profile.isAdmin, let counts = try? await api.requestCounts() { pendingCount = counts.pending }
        }
        .id(account.id)
    }

    private var api: TipsarrAPI { TipsarrAPI(serverURL: account.serverURL, token: account.token) }
}

private struct TabPlaceholder: View {
    let title: LocalizedStringResource
    let symbol: String

    var body: some View {
        NavigationStack {
            StateView(symbol: symbol, title: title)
                .background(Tokens.palette.bg)
                .navigationTitle(Text(title))
        }
    }
}
