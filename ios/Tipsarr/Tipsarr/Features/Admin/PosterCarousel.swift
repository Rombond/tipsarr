import SwiftUI

/// One title in a carousel or a list: a poster, a title and a short note.
struct PosterRow: Hashable, Identifiable {
    var id: String
    var title: String
    var note: String
    var posterPath: String?
    /// True when `posterPath` is a path on the Tipsarr server (Jellyfin posters) instead of a TMDB path.
    var onServer = false
    var route: MediaRoute
}

/// Carries a whole list to its full-screen view.
struct PosterListRoute: Hashable {
    var title: String
    var rows: [PosterRow]
    /// Show the position (1, 2, 3…) in front of each row.
    var ranked = false
}

/// A row of posters in a carousel with a See all button (the list view, or any other route).
struct PosterCarousel: View {
    let title: LText
    /// Catalog key of the list screen title.
    let titleKey: String
    let rows: [PosterRow]
    var ranked = false
    /// Where See all goes; the full list when nil.
    var seeAll: ProfileRoute?

    var body: some View {
        if !rows.isEmpty {
            VStack(alignment: .leading, spacing: Tokens.Spacing.md) {
                HStack {
                    Text(title).font(.title3.weight(.bold)).accessibilityAddTraits(.isHeader)
                    Spacer()
                    NavigationLink(value: seeAll ?? .posterList(PosterListRoute(title: titleKey, rows: rows, ranked: ranked))) {
                        Text("m.discover.see_all").font(.subheadline)
                    }
                    .foregroundStyle(Tokens.palette.mutedFg)
                }
                ScrollView(.horizontal, showsIndicators: false) {
                    LazyHStack(alignment: .top, spacing: Tokens.Spacing.md) {
                        ForEach(rows) { row in
                            NavigationLink(value: row.route) { card(row) }.buttonStyle(.plain)
                        }
                    }
                }
                .scrollClipDisabled()
            }
        }
    }

    private func card(_ row: PosterRow) -> some View {
        VStack(alignment: .leading, spacing: Tokens.Spacing.xs) {
            PosterRowImage(row: row)
                .aspectRatio(1 / Tokens.Size.posterRatio, contentMode: .fit)
                .background(Tokens.palette.muted)
                .clipShape(.rect(cornerRadius: Tokens.Radius.md))
            Text(verbatim: row.title).font(.subheadline.weight(.medium)).lineLimit(1)
            Text(verbatim: row.note).font(.caption).foregroundStyle(Tokens.palette.mutedFg).lineLimit(1)
        }
        .frame(width: 120)
        .accessibilityElement(children: .ignore)
        .accessibilityLabel(Text(verbatim: "\(row.title), \(row.note)"))
    }
}

struct PosterRowImage: View {
    let row: PosterRow

    var body: some View {
        if row.onServer {
            RemoteImage(serverPath: row.posterPath) { placeholder }
        } else {
            RemoteImage(path: row.posterPath, size: .w342) { placeholder }
        }
    }

    private var placeholder: some View {
        ZStack { Tokens.palette.muted; Image(systemName: "film").foregroundStyle(Tokens.palette.mutedFg) }
    }
}

/// The same list as rows.
struct PosterList: View {
    let route: PosterListRoute

    var body: some View {
        List {
            ForEach(Array(route.rows.enumerated()), id: \.element.id) { index, row in
                NavigationLink(value: row.route) {
                    HStack(spacing: Tokens.Spacing.md) {
                        if route.ranked {
                            Text(verbatim: String(index + 1)).font(.headline).foregroundStyle(Tokens.palette.mutedFg).frame(width: 28)
                        }
                        PosterRowImage(row: row)
                            .frame(width: 44, height: 66)
                            .background(Tokens.palette.muted)
                            .clipShape(.rect(cornerRadius: 6))
                            .accessibilityHidden(true)
                        VStack(alignment: .leading, spacing: 2) {
                            Text(verbatim: row.title).font(.body.weight(.medium)).lineLimit(2)
                            if !row.note.isEmpty { Text(verbatim: row.note).font(.footnote).foregroundStyle(Tokens.palette.mutedFg) }
                        }
                    }
                    .accessibilityElement(children: .combine)
                }
                .listRowBackground(Color.clear)
            }
        }
        .listStyle(.plain)
        .background(Tokens.palette.bg)
        .navigationTitle(Text(LText(stringLiteral: route.title)))
        .navigationBarTitleDisplayMode(.inline)
    }
}

enum StatsFormat {
    static func hours(_ value: Double) -> String {
        let unit = L10n.string("stats.unit_h")
        return value >= 100 ? "\(Int(value.rounded())) \(unit)" : "\(value.formatted(.number.precision(.fractionLength(1)))) \(unit)"
    }

    /// "12.5 h · 8 episodes watched" for shows, "3.2 h · 2 plays" for movies.
    static func note(_ item: StatsTop) -> String {
        let count = item.type == .tv ? L10n.string("stats.n_episodes", String(item.plays)) : L10n.string("stats.n_plays", String(item.plays))
        return "\(hours(item.hours)) · \(count)"
    }
}

extension StatsTop {
    var row: PosterRow {
        PosterRow(id: id, title: title, note: StatsFormat.note(self), posterPath: posterPath, onServer: true, route: route)
    }
}

extension RequestRecord {
    /// "Asked for, never watched": since when the title is available for the requester.
    var unwatchedRow: PosterRow {
        PosterRow(id: id, title: title, note: L10n.string("stats.unwatched_since", createdAt.formatted(date: .abbreviated, time: .omitted)),
                  posterPath: posterPath, route: MediaRoute(type: type, tmdbId: tmdbId, title: title))
    }
}

extension LibraryItem {
    var row: PosterRow {
        PosterRow(id: id, title: title, note: year.map(String.init) ?? "", posterPath: posterPath, onServer: true, route: route)
    }
}
