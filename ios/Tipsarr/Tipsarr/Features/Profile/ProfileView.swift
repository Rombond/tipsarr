import SwiftUI

struct ProfileView: View {
    @Environment(SessionManager.self) private var session
    let account: Account
    let profile: Profile
    @AppStorage("avatarVersion") private var avatarVersion = 0
    @State private var showPicture = false

    var body: some View {
        NavigationStack {
            List {
                Section {
                    HStack(spacing: Tokens.Spacing.lg) {
                        Button { showPicture = true } label: {
                            AvatarView(userID: profile.id, name: profile.name, size: 72, version: avatarVersion)
                                .overlay(alignment: .bottomTrailing) {
                                    Image(systemName: "camera.fill")
                                        .font(.caption2)
                                        .foregroundStyle(Tokens.palette.primaryFg)
                                        .padding(6)
                                        .background(Tokens.palette.primary, in: .circle)
                                }
                        }
                        .buttonStyle(.plain)
                        .accessibilityLabel(Text("profile.change_picture"))
                        VStack(alignment: .leading, spacing: Tokens.Spacing.xs) {
                            Text(verbatim: profile.name).font(.title2.weight(.bold))
                            Text(profile.isAdmin ? "nav.administrator" : "nav.member").font(.subheadline).foregroundStyle(Tokens.palette.mutedFg)
                            Text(verbatim: account.serverURL.host() ?? account.serverURL.absoluteString)
                                .font(.footnote).foregroundStyle(Tokens.palette.mutedFg)
                        }
                    }
                    .listRowBackground(Color.clear)
                }
                Section {
                    NavigationLink(value: ProfileRoute.watchlist) { Label { Text("watchlist.title") } icon: { Image(systemName: "bookmark") } }
                    NavigationLink(value: ProfileRoute.hidden) { Label { Text("profile.hidden_title") } icon: { Image(systemName: "eye.slash") } }
                }
                Section {
                    NavigationLink(value: ProfileRoute.settings) { Label { Text("nav.settings") } icon: { Image(systemName: "gearshape") } }
                }
            }
            .scrollContentBackground(.hidden)
            .background(Tokens.palette.bg)
            .navigationTitle("profile.title")
            .navigationDestination(for: ProfileRoute.self) { route in
                switch route {
                case .watchlist: WatchlistScreen()
                case .hidden: HiddenScreen()
                case .settings: SettingsView(account: account, profile: profile)
                case .devices: DevicesView()
                case .accounts: AccountsView()
                }
            }
            .mediaDestinations()
            .sheet(isPresented: $showPicture) {
                PictureSheet(profile: profile) { avatarVersion += 1 }
                    .tipsarrSheet(detents: [.medium])
            }
        }
    }
}

enum ProfileRoute: Hashable { case watchlist, hidden, settings, devices, accounts }
