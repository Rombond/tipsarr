import SwiftUI

struct ProfileView: View {
    let account: Account
    let profile: Profile
    @Binding var path: NavigationPath
    var openRequests: () -> Void = {}

    @Environment(\.appContext) private var context
    @Environment(\.liveUpdates) private var live
    @Environment(\.detailPane) private var pane
    @AppStorage("avatarVersion") var avatarVersion = 0
    @Environment(SessionManager.self) var session
    @State var model: ProfileModel
    /// Width of the content area; picks the compact, portrait-iPad or landscape-iPad layout.
    @State private var width: CGFloat = 0

    init(account: Account, profile: Profile, api: TipsarrAPI, path: Binding<NavigationPath>, openRequests: @escaping () -> Void = {}) {
        self.account = account
        _path = path
        self.profile = profile
        self.openRequests = openRequests
        let model = ProfileModel(api: api)
        model.isAdmin = profile.isAdmin
        _model = State(initialValue: model)
    }

    var body: some View {
        NavigationStack(path: $path) {
            ScrollView {
                Group {
                    switch width == 0 ? ContentWidth.compact : ContentWidth(width) {
                    case .compact: compactBody
                    case .regular: portraitBody
                    case .wide: landscapeBody
                    }
                }
                .padding(Tokens.Spacing.lg)
                .frame(maxWidth: width == 0 || width < 620 ? 720 : 1100)
                .frame(maxWidth: .infinity)
            }
            .onGeometryChange(for: CGFloat.self, of: { $0.size.width }) { width = $0 }
            .paneInset()
            .background(Tokens.palette.bg)
            .navigationTitle("profile.title")
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    NavigationLink(value: ProfileRoute.settings) { Image(systemName: "gearshape") }
                        .accessibilityLabel(Text("nav.settings"))
                }
            }
            .navigationDestination(for: ProfileRoute.self) { ProfileDestination(route: $0, account: account, profile: profile).paneInset() }
            .mediaDestinations()
            // Back on the Profile root (Duo): the right pane returns to the statistics.
            .onAppear { pane?.content = nil }
            .task { await model.load() }
            .refreshable { await model.load() }
            .onChange(of: live?.requestsTick) { Task { await model.load() } }
        }
    }

    /// iPhone and narrow windows: one centred column.
    private var compactBody: some View {
        VStack(spacing: Tokens.Spacing._2xl) {
            header
            stats
            // See all goes to the full Stats page.
            PosterCarousel(title: "stats.top", titleKey: "stats.top", rows: model.topWatched.map(\.row), ranked: true, seeAll: .stats)
            if profile.isAdmin { AdminGroup(openIssues: model.openIssues) }
            recentRequests
        }
    }

    // MARK: Header

    private var header: some View {
        VStack(spacing: Tokens.Spacing.md) {
            AvatarView(userID: profile.id, name: profile.name, size: 88, version: avatarVersion)
            Text(verbatim: profile.name).font(.title.weight(.bold))
            if let line = memberLine {
                Text(verbatim: line).font(.footnote).foregroundStyle(Tokens.palette.mutedFg).multilineTextAlignment(.center)
            }
        }
        .frame(maxWidth: .infinity)
    }

    /// "Member since Oct 2025 · last seen today"
    var memberLine: String? {
        var parts: [String] = []
        if let created = profile.createdAt, created.timeIntervalSince1970 > 0 {
            parts.append(L10n.string("profile.member_since", created.formatted(.dateTime.month(.abbreviated).year())))
        }
        if let seen = profile.lastLoginAt, seen.timeIntervalSince1970 > 0 {
            parts.append(L10n.string("profile.last_seen", seen.formatted(.relative(presentation: .named))))
        }
        return parts.isEmpty ? nil : parts.joined(separator: " · ")
    }

    // MARK: Stats

    private var stats: some View {
        HStack(spacing: Tokens.Spacing.md) {
            StatCard(symbol: "checklist", value: model.requestCount, label: "profile.stat_requests") { openRequests() }
            ProfileLink(route: .watchlist) {
                StatTile(symbol: "bookmark", value: model.watchlistCount, label: "profile.stat_watchlist")
            }
            .buttonStyle(.plain)
            StatCard(symbol: "eye", value: model.watchedCount, label: "profile.stat_watched")
        }
    }

    // MARK: Recent requests

    @ViewBuilder private var recentRequests: some View {
        if !model.recent.isEmpty {
            VStack(alignment: .leading, spacing: Tokens.Spacing.md) {
                HStack {
                    Text("profile.recent_requests").font(.title3.weight(.bold)).accessibilityAddTraits(.isHeader)
                    Spacer()
                    Button(action: openRequests) { Text("requests.tab.all").font(.subheadline) }
                        .foregroundStyle(Tokens.palette.mutedFg)
                }
                VStack(spacing: 0) {
                    ForEach(model.recent.prefix(3)) { record in
                        MediaLink(route: MediaRoute(type: record.type, tmdbId: record.tmdbId, title: record.title)) {
                            HStack(spacing: Tokens.Spacing.md) {
                                RemoteImage(path: record.posterPath, size: .w92) {
                                    Image(systemName: "film").foregroundStyle(Tokens.palette.mutedFg)
                                }
                                .frame(width: 44, height: 66)
                                .background(Tokens.palette.muted)
                                .clipShape(.rect(cornerRadius: Tokens.Radius.sm - 2))
                                .accessibilityHidden(true)
                                VStack(alignment: .leading, spacing: Tokens.Spacing.xs) {
                                    Text(verbatim: record.title).font(.body.weight(.semibold)).lineLimit(1)
                                    StatusBadge(state: record.state)
                                }
                                Spacer(minLength: 0)
                            }
                            .padding(.vertical, Tokens.Spacing.sm)
                            .contentShape(.rect)
                            .accessibilityElement(children: .combine)
                        }
                        .buttonStyle(.plain)
                        Divider()
                    }
                }
            }
        }
    }
}

enum ProfileRoute: Hashable { case watchlist, hidden, settings, devices, accounts, appIcon, issues, issue(String), users, user(String), sync, stats, posterList(PosterListRoute) }

/// Number with an icon and a label.
struct StatTile: View {
    let symbol: String
    let value: Int?
    let label: LText
    /// Icon beside the number (wide tiles of the portrait iPad layout) instead of above it.
    var horizontal = false

    var body: some View {
        Group {
            if horizontal {
                HStack(spacing: Tokens.Spacing.md) {
                    Image(systemName: symbol)
                        .foregroundStyle(Tokens.palette.fg)
                        .frame(width: 40, height: 40)
                        .background(Tokens.palette.muted, in: .rect(cornerRadius: Tokens.Radius.md))
                    VStack(alignment: .leading, spacing: 2) {
                        Text(verbatim: value.map(String.init) ?? "–").font(.title.weight(.bold))
                        Text(label).font(.footnote).foregroundStyle(Tokens.palette.mutedFg)
                    }
                    Spacer(minLength: 0)
                }
                .padding(Tokens.Spacing.md)
            } else {
                VStack(spacing: Tokens.Spacing.xs) {
                    Image(systemName: symbol).foregroundStyle(Tokens.palette.mutedFg)
                    Text(verbatim: value.map(String.init) ?? "–").font(.title2.weight(.bold))
                    Text(label).font(.footnote).foregroundStyle(Tokens.palette.mutedFg)
                }
                .frame(maxWidth: .infinity)
                .padding(.vertical, Tokens.Spacing.md)
            }
        }
        .frame(maxWidth: .infinity)
        .background(Tokens.palette.card, in: .rect(cornerRadius: Tokens.Radius.md))
        .overlay { RoundedRectangle(cornerRadius: Tokens.Radius.md).strokeBorder(Tokens.palette.border) }
        .accessibilityElement(children: .combine)
    }
}

struct StatCard: View {
    let symbol: String
    let value: Int?
    let label: LText
    var horizontal = false
    var action: (() -> Void)?

    var body: some View {
        if let action {
            Button(action: action) { StatTile(symbol: symbol, value: value, label: label, horizontal: horizontal) }.buttonStyle(.plain)
        } else {
            StatTile(symbol: symbol, value: value, label: label, horizontal: horizontal)
        }
    }
}

/// Administrator tools: one grouped card of rows, like the Administration group in Penpot.
private struct AdminGroup: View {
    let openIssues: Int?

    var body: some View {
        VStack(alignment: .leading, spacing: Tokens.Spacing.sm) {
            Text("m.admin.title").font(.footnote.weight(.semibold)).foregroundStyle(Tokens.palette.mutedFg).padding(.horizontal, Tokens.Spacing.xs)
            VStack(spacing: 0) {
                ProfileLink(route: .issues) { row("exclamationmark.bubble", Tokens.Status.failed, "nav.issues", openIssues) }
                    .buttonStyle(.plain)
                Divider().padding(.leading, 52)
                ProfileLink(route: .users) { row("person.2", Tokens.Status.approved, "nav.users", nil) }
                    .buttonStyle(.plain)
                Divider().padding(.leading, 52)
                ProfileLink(route: .sync) { row("arrow.triangle.2.circlepath", Tokens.Status.searching, "m.admin.sync", nil) }
                    .buttonStyle(.plain)
            }
            .background(Tokens.palette.card, in: .rect(cornerRadius: Tokens.Radius.md))
            .overlay { RoundedRectangle(cornerRadius: Tokens.Radius.md).strokeBorder(Tokens.palette.border) }
        }
    }

    private func row(_ symbol: String, _ color: Color, _ title: LText, _ value: Int?) -> some View {
        HStack(spacing: Tokens.Spacing.md) {
            Image(systemName: symbol)
                .font(.subheadline.weight(.semibold))
                .foregroundStyle(color)
                .frame(width: 30, height: 30)
                .background(color.opacity(0.16), in: .rect(cornerRadius: Tokens.Radius.sm - 2))
            Text(title).font(.body)
            Spacer(minLength: 0)
            if let value, value > 0 { Text(verbatim: String(value)).font(.subheadline).foregroundStyle(Tokens.palette.mutedFg) }
            Image(systemName: "chevron.right").font(.footnote.weight(.semibold)).foregroundStyle(Tokens.palette.mutedFg)
        }
        .padding(.horizontal, Tokens.Spacing.md)
        .frame(minHeight: Tokens.Size.touchTarget + Tokens.Spacing.sm)
        .contentShape(.rect)
    }
}

/// The screen of a `ProfileRoute`; used by the Profile stack and by the Duo's right pane.
struct ProfileDestination: View {
    let route: ProfileRoute
    let account: Account
    let profile: Profile

    var body: some View {
        switch route {
        case .watchlist: WatchlistScreen()
        case .hidden: HiddenScreen()
        case .settings: SettingsView(account: account, profile: profile)
        case .devices: DevicesView()
        case .accounts: AccountsView()
        case .appIcon: AppIconPicker()
        case .issues: IssuesView()
        case .issue(let id): IssueDetailView(id: id)
        case .users: UsersView()
        case .user(let id): UserDetailView(id: id)
        case .sync: SyncView()
        case .stats: StatsView()
        case .posterList(let route): PosterList(route: route)
        }
    }
}
