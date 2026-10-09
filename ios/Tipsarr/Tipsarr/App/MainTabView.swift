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

    var body: some View {
        TabView(selection: $selection) {
            Tab("m.tab.discover", systemImage: "safari", value: AppTab.discover) {
                DiscoverView(api: api) { selection = .search }
            }
            Tab("m.tab.search", systemImage: "magnifyingglass", value: AppTab.search, role: .search) {
                SearchView(api: api)
            }
            Tab("m.tab.requests", systemImage: "checklist", value: AppTab.requests) {
                TabPlaceholder(title: "m.tab.requests", symbol: "checklist")
            }
            Tab("m.tab.library", systemImage: "books.vertical", value: AppTab.library) {
                TabPlaceholder(title: "m.tab.library", symbol: "books.vertical")
            }
            Tab("m.tab.profile", systemImage: "person.crop.circle", value: AppTab.profile) {
                ProfilePlaceholder(account: account, profile: profile)
            }
        }
        .environment(\.imageSource, ImageSource(serverURL: account.serverURL, token: account.token))
        .environment(\.appContext, AppContext(api: api, profile: profile, userFolderChoice: session.serverStatus?.userFolderChoice ?? false))
        .environment(toast)
        .modifier(ToastOverlay(center: toast))
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

/// Replaced by the Profile screen in step 6.
private struct ProfilePlaceholder: View {
    @Environment(SessionManager.self) private var session
    let account: Account
    let profile: Profile

    var body: some View {
        NavigationStack {
            VStack(spacing: Tokens.Spacing.lg) {
                Text(verbatim: profile.name).font(.title.weight(.bold))
                Text(verbatim: account.serverURL.absoluteString).foregroundStyle(Tokens.palette.mutedFg)
                Button("m.settings.sign_out_confirm") { Task { await session.signOut(account) } }
                    .buttonStyle(.tipsarr(.secondary))
            }
            .padding()
            .frame(maxWidth: .infinity, maxHeight: .infinity)
            .background(Tokens.palette.bg)
            .navigationTitle("m.tab.profile")
        }
    }
}
