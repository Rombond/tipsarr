import SwiftUI

struct ProfileView: View {
    let account: Account
    let profile: Profile
    var openRequests: () -> Void = {}
    var openPending: () -> Void = {}

    @Environment(\.appContext) private var context
    @Environment(\.liveUpdates) private var live
    @AppStorage("avatarVersion") private var avatarVersion = 0
    @State private var model: ProfileModel
    @State private var showPicture = false

    init(account: Account, profile: Profile, api: TipsarrAPI, openRequests: @escaping () -> Void = {}, openPending: @escaping () -> Void = {}) {
        self.account = account
        self.profile = profile
        self.openRequests = openRequests
        self.openPending = openPending
        let model = ProfileModel(api: api)
        model.isAdmin = profile.isAdmin
        _model = State(initialValue: model)
    }

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(spacing: Tokens.Spacing._2xl) {
                    header
                    stats
                    if !profile.isAdmin {
                        NavigationLink(value: ProfileRoute.stats) {
                            HStack {
                                Label { Text("nav.stats") } icon: { Image(systemName: "chart.bar") }
                                Spacer()
                                Image(systemName: "chevron.right").font(.footnote.weight(.semibold)).foregroundStyle(Tokens.palette.mutedFg)
                            }
                            .padding(.horizontal, Tokens.Spacing.md)
                            .frame(minHeight: Tokens.Size.touchTarget + Tokens.Spacing.sm)
                            .background(Tokens.palette.card, in: .rect(cornerRadius: Tokens.Radius.md))
                            .overlay { RoundedRectangle(cornerRadius: Tokens.Radius.md).strokeBorder(Tokens.palette.border) }
                        }
                        .buttonStyle(.plain)
                    }
                    if profile.isAdmin { AdminGroup(pendingCount: model.pendingCount, openIssues: model.openIssues, openRequests: openPending) }
                    recentRequests
                }
                .padding(Tokens.Spacing.lg)
                .frame(maxWidth: 720)
                .frame(maxWidth: .infinity)
            }
            .background(Tokens.palette.bg)
            .navigationTitle("profile.title")
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    NavigationLink(value: ProfileRoute.settings) { Image(systemName: "gearshape") }
                        .accessibilityLabel(Text("nav.settings"))
                }
            }
            .navigationDestination(for: ProfileRoute.self) { route in
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
                }
            }
            .mediaDestinations()
            .task { await model.load() }
            .refreshable { await model.load() }
            .onChange(of: live?.requestsTick) { Task { await model.load() } }
            .sheet(isPresented: $showPicture) {
                PictureSheet(profile: profile) { avatarVersion += 1 }
                    .tipsarrSheet(detents: [.medium])
            }
        }
    }

    // MARK: Header

    private var header: some View {
        VStack(spacing: Tokens.Spacing.md) {
            Button { showPicture = true } label: {
                AvatarView(userID: profile.id, name: profile.name, size: 88, version: avatarVersion)
                    .overlay(alignment: .bottomTrailing) {
                        Image(systemName: "camera.fill")
                            .font(.caption2)
                            .foregroundStyle(Tokens.palette.primaryFg)
                            .padding(7)
                            .background(Tokens.palette.primary, in: .circle)
                    }
            }
            .buttonStyle(.plain)
            .accessibilityLabel(Text("profile.change_picture"))
            Text(verbatim: profile.name).font(.title.weight(.bold))
            if let line = memberLine {
                Text(verbatim: line).font(.footnote).foregroundStyle(Tokens.palette.mutedFg).multilineTextAlignment(.center)
            }
        }
        .frame(maxWidth: .infinity)
    }

    /// "Member since Oct 2025 · last seen today"
    private var memberLine: String? {
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
            NavigationLink(value: ProfileRoute.watchlist) {
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
                    ForEach(model.recent) { record in
                        NavigationLink(value: MediaRoute(type: record.type, tmdbId: record.tmdbId, title: record.title)) {
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

enum ProfileRoute: Hashable { case watchlist, hidden, settings, devices, accounts, appIcon, issues, issue(String), users, user(String), sync, stats }

/// Number with an icon and a label.
private struct StatTile: View {
    let symbol: String
    let value: Int?
    let label: LText

    var body: some View {
        VStack(spacing: Tokens.Spacing.xs) {
            Image(systemName: symbol).foregroundStyle(Tokens.palette.mutedFg)
            Text(verbatim: value.map(String.init) ?? "–").font(.title2.weight(.bold))
            Text(label).font(.footnote).foregroundStyle(Tokens.palette.mutedFg)
        }
        .frame(maxWidth: .infinity)
        .padding(.vertical, Tokens.Spacing.md)
        .background(Tokens.palette.card, in: .rect(cornerRadius: Tokens.Radius.md))
        .overlay { RoundedRectangle(cornerRadius: Tokens.Radius.md).strokeBorder(Tokens.palette.border) }
        .accessibilityElement(children: .combine)
    }
}

private struct StatCard: View {
    let symbol: String
    let value: Int?
    let label: LText
    var action: (() -> Void)?

    var body: some View {
        if let action {
            Button(action: action) { StatTile(symbol: symbol, value: value, label: label) }.buttonStyle(.plain)
        } else {
            StatTile(symbol: symbol, value: value, label: label)
        }
    }
}

/// Administrator tools: one grouped card of rows, like the Administration group in Penpot.
private struct AdminGroup: View {
    let pendingCount: Int?
    let openIssues: Int?
    let openRequests: () -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: Tokens.Spacing.sm) {
            Text("m.admin.title").font(.footnote.weight(.semibold)).foregroundStyle(Tokens.palette.mutedFg).padding(.horizontal, Tokens.Spacing.xs)
            VStack(spacing: 0) {
                Button(action: openRequests) { row("clock", Tokens.Status.requested, "m.admin.pending_requests", pendingCount) }
                    .buttonStyle(.plain)
                Divider().padding(.leading, 52)
                NavigationLink(value: ProfileRoute.issues) { row("exclamationmark.bubble", Tokens.Status.failed, "nav.issues", openIssues) }
                    .buttonStyle(.plain)
                Divider().padding(.leading, 52)
                NavigationLink(value: ProfileRoute.users) { row("person.2", Tokens.Status.approved, "nav.users", nil) }
                    .buttonStyle(.plain)
                Divider().padding(.leading, 52)
                NavigationLink(value: ProfileRoute.sync) { row("arrow.triangle.2.circlepath", Tokens.Status.searching, "m.admin.sync", nil) }
                    .buttonStyle(.plain)
                Divider().padding(.leading, 52)
                NavigationLink(value: ProfileRoute.stats) { row("chart.bar", Tokens.Status.downloading, "nav.stats", nil) }
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
