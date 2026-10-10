import SwiftUI

/// Profile on a wide screen: the two iPad layouts of Penpot page 10.
/// Portrait: one wide column. Landscape: a profile column on the left, the dashboard on the right.
extension ProfileView {
    // MARK: Portrait (regular width)

    var portraitBody: some View {
        VStack(alignment: .leading, spacing: Tokens.Spacing._2xl) {
            headerCard(vertical: false)
            statTiles(vertical: false)
            mostWatched(posterWidth: 116)
            recentGrid
            if profile.isAdmin { AdminTiles(openIssues: model.openIssues) }
        }
    }

    // MARK: Landscape (wide)

    var landscapeBody: some View {
        HStack(alignment: .top, spacing: 28) {
            VStack(spacing: Tokens.Spacing.lg) {
                headerCard(vertical: true)
                statTiles(vertical: true)
                NavigationLink(value: ProfileRoute.settings) {
                    HStack(spacing: Tokens.Spacing.md) {
                        Image(systemName: "gearshape").frame(width: 24)
                        Text("nav.settings").font(.body.weight(.medium))
                        Spacer()
                        Image(systemName: "chevron.right").font(.footnote.weight(.semibold)).foregroundStyle(Tokens.palette.mutedFg)
                    }
                    .padding(.horizontal, Tokens.Spacing.lg)
                    .frame(minHeight: 56)
                    .background(Tokens.palette.card, in: .rect(cornerRadius: Tokens.Radius.md))
                    .overlay { RoundedRectangle(cornerRadius: Tokens.Radius.md).strokeBorder(Tokens.palette.border) }
                }
                .buttonStyle(.plain)
            }
            .frame(width: 300)
            VStack(alignment: .leading, spacing: Tokens.Spacing._2xl) {
                mostWatched(posterWidth: 114)
                recentGrid
                if profile.isAdmin { AdminTiles(openIssues: model.openIssues) }
            }
            .frame(maxWidth: .infinity)
        }
    }

    // MARK: Pieces

    func mostWatched(posterWidth: CGFloat) -> some View {
        PosterCarousel(title: "stats.top", titleKey: "stats.top", rows: model.topWatched.map(\.row), ranked: true, seeAll: .stats, posterWidth: posterWidth)
    }

    /// Profile card: avatar, name, role, dates and the server. Vertical in the landscape column, horizontal in portrait.
    @ViewBuilder func headerCard(vertical: Bool) -> some View {
        let avatar = AvatarView(userID: profile.id, name: profile.name, size: vertical ? 104 : 96, version: avatarVersion)
        let server = VStack(alignment: vertical ? .leading : .leading, spacing: 2) {
            Label { Text(verbatim: account.serverURL.host() ?? account.serverURL.absoluteString).font(.subheadline.weight(.semibold)) }
                icon: { Image(systemName: "server.rack").foregroundStyle(Tokens.palette.mutedFg) }
            if let version = session.serverStatus?.version {
                Text(verbatim: L10n.string("m.settings.server_version", version)).font(.caption).foregroundStyle(Tokens.palette.mutedFg)
            }
        }
        Group {
            if vertical {
                VStack(spacing: Tokens.Spacing.md) {
                    avatar
                    Text(verbatim: profile.name).font(.title.weight(.bold))
                    Text(profile.isAdmin ? "nav.administrator" : "nav.member").font(.subheadline).foregroundStyle(Tokens.palette.mutedFg)
                    if let line = memberLine { Text(verbatim: line).font(.footnote).foregroundStyle(Tokens.palette.mutedFg).multilineTextAlignment(.center) }
                    Divider()
                    server.frame(maxWidth: .infinity, alignment: .leading)
                }
                .padding(Tokens.Spacing.xl)
            } else {
                HStack(spacing: Tokens.Spacing.xl) {
                    avatar
                    VStack(alignment: .leading, spacing: Tokens.Spacing.sm) {
                        Text(verbatim: profile.name).font(.title.weight(.bold))
                        Text(verbatim: [L10n.string(profile.isAdmin ? "nav.administrator" : "nav.member"), memberLine].compactMap { $0 }.joined(separator: " · "))
                            .font(.subheadline).foregroundStyle(Tokens.palette.mutedFg)
                        server
                    }
                    Spacer(minLength: 0)
                }
                .padding(Tokens.Spacing.xl)
            }
        }
        .frame(maxWidth: .infinity)
        .background(Tokens.palette.card, in: .rect(cornerRadius: Tokens.Radius.xl))
        .overlay { RoundedRectangle(cornerRadius: Tokens.Radius.xl).strokeBorder(Tokens.palette.border) }
    }

    func statTiles(vertical: Bool) -> some View {
        HStack(spacing: vertical ? Tokens.Spacing.md : Tokens.Spacing.lg) {
            StatCard(symbol: "checklist", value: model.requestCount, label: "profile.stat_requests", horizontal: !vertical) { openRequests() }
            ProfileLink(route: .watchlist) {
                StatTile(symbol: "bookmark", value: model.watchlistCount, label: "profile.stat_watchlist", horizontal: !vertical)
            }
            .buttonStyle(.plain)
            StatCard(symbol: "eye", value: model.watchedCount, label: "profile.stat_watched", horizontal: !vertical)
        }
    }

    /// Recent requests as cards, two per row.
    @ViewBuilder var recentGrid: some View {
        if !model.recent.isEmpty {
            VStack(alignment: .leading, spacing: Tokens.Spacing.md) {
                HStack {
                    Text("profile.recent_requests").font(.title3.weight(.bold)).accessibilityAddTraits(.isHeader)
                    Spacer()
                    Button(action: openRequests) { Text("requests.tab.all").font(.subheadline) }.foregroundStyle(Tokens.palette.mutedFg)
                }
                LazyVGrid(columns: [GridItem(.flexible(), spacing: 14), GridItem(.flexible())], spacing: 12) {
                    ForEach(model.recent.prefix(4)) { record in
                        MediaLink(route: MediaRoute(type: record.type, tmdbId: record.tmdbId, title: record.title)) {
                            RequestCardTile(record: record)
                        }
                        .buttonStyle(.plain)
                    }
                }
            }
        }
    }
}

struct RequestCardTile: View {
    let record: RequestRecord

    var body: some View {
        HStack(spacing: Tokens.Spacing.md) {
            RemoteImage(path: record.posterPath, size: .w92) { Image(systemName: "film").foregroundStyle(Tokens.palette.mutedFg) }
                .frame(width: 44, height: 66)
                .background(Tokens.palette.muted)
                .clipShape(.rect(cornerRadius: 6))
                .accessibilityHidden(true)
            VStack(alignment: .leading, spacing: Tokens.Spacing.xs) {
                Text(verbatim: record.title).font(.body.weight(.semibold)).lineLimit(1)
                StatusBadge(state: record.state)
                Text(record.createdAt, format: .relative(presentation: .named)).font(.caption).foregroundStyle(Tokens.palette.mutedFg)
            }
            Spacer(minLength: 0)
        }
        .padding(Tokens.Spacing.md)
        .background(Tokens.palette.card, in: .rect(cornerRadius: Tokens.Radius.md))
        .overlay { RoundedRectangle(cornerRadius: Tokens.Radius.md).strokeBorder(Tokens.palette.border) }
        .accessibilityElement(children: .combine)
    }
}

/// The administrator tools as four tiles in a row (wide screens); the iPhone has the list card instead.
struct AdminTiles: View {
    let openIssues: Int?

    var body: some View {
        VStack(alignment: .leading, spacing: Tokens.Spacing.sm) {
            Text("m.admin.title").font(.footnote.weight(.semibold)).foregroundStyle(Tokens.palette.mutedFg).textCase(.uppercase)
            HStack(spacing: 14) {
                tile("exclamationmark.bubble", Tokens.Status.failed, "nav.issues", .issues, openIssues)
                tile("person.2", Tokens.Status.approved, "nav.users", .users, nil)
                tile("arrow.triangle.2.circlepath", Tokens.Status.searching, "m.admin.sync", .sync, nil)
                tile("chart.bar", Tokens.Status.downloading, "nav.stats", .stats, nil)
            }
        }
    }

    private func tile(_ symbol: String, _ color: Color, _ title: LText, _ route: ProfileRoute, _ count: Int?) -> some View {
        ProfileLink(route: route) {
            VStack(alignment: .leading, spacing: Tokens.Spacing.md) {
                HStack {
                    Image(systemName: symbol).foregroundStyle(color)
                        .frame(width: 34, height: 34)
                        .background(color.opacity(0.16), in: .rect(cornerRadius: 10))
                    Spacer()
                    if let count, count > 0 { Text(verbatim: String(count)).font(.title3.weight(.bold)) }
                }
                Text(title).font(.subheadline.weight(.semibold)).lineLimit(1).minimumScaleFactor(0.8)
            }
            .padding(Tokens.Spacing.md)
            .frame(maxWidth: .infinity, alignment: .leading)
            .background(Tokens.palette.card, in: .rect(cornerRadius: Tokens.Radius.md))
            .overlay { RoundedRectangle(cornerRadius: Tokens.Radius.md).strokeBorder(Tokens.palette.border) }
        }
        .buttonStyle(.plain)
    }
}
