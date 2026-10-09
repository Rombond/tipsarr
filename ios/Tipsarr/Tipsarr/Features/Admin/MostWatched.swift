import SwiftUI

/// Carries a whole "most watched" list to its full-screen view.
struct TopListRoute: Hashable {
    var title: String
    var items: [StatsTop]
}

/// "Most watched" row: every title as a poster in a carousel, with a See all button for the list view.
struct MostWatchedCarousel: View {
    let title: LText
    /// Catalog key of the list screen title.
    let titleKey: String
    let items: [StatsTop]

    var body: some View {
        if !items.isEmpty {
            VStack(alignment: .leading, spacing: Tokens.Spacing.md) {
                HStack {
                    Text(title).font(.title3.weight(.bold)).accessibilityAddTraits(.isHeader)
                    Spacer()
                    NavigationLink(value: ProfileRoute.topList(TopListRoute(title: titleKey, items: items))) {
                        Text("m.discover.see_all").font(.subheadline)
                    }
                    .foregroundStyle(Tokens.palette.mutedFg)
                }
                ScrollView(.horizontal, showsIndicators: false) {
                    LazyHStack(alignment: .top, spacing: Tokens.Spacing.md) {
                        ForEach(items) { item in
                            NavigationLink(value: item.route) { card(item) }.buttonStyle(.plain)
                        }
                    }
                }
                .scrollClipDisabled()
            }
        }
    }

    private func card(_ item: StatsTop) -> some View {
        VStack(alignment: .leading, spacing: Tokens.Spacing.xs) {
            RemoteImage(serverPath: item.posterPath) {
                ZStack { Tokens.palette.muted; Image(systemName: "film").foregroundStyle(Tokens.palette.mutedFg) }
            }
            .aspectRatio(1 / Tokens.Size.posterRatio, contentMode: .fit)
            .background(Tokens.palette.muted)
            .clipShape(.rect(cornerRadius: Tokens.Radius.md))
            Text(verbatim: item.title).font(.subheadline.weight(.medium)).lineLimit(1)
            Text(verbatim: StatsFormat.note(item)).font(.caption).foregroundStyle(Tokens.palette.mutedFg).lineLimit(1)
        }
        .frame(width: 120)
        .accessibilityElement(children: .ignore)
        .accessibilityLabel(Text(verbatim: "\(item.title), \(StatsFormat.note(item))"))
    }
}

/// The same list as rows.
struct MostWatchedList: View {
    let route: TopListRoute

    var body: some View {
        List {
            ForEach(Array(route.items.enumerated()), id: \.element.id) { index, item in
                NavigationLink(value: item.route) {
                    HStack(spacing: Tokens.Spacing.md) {
                        Text(verbatim: String(index + 1)).font(.headline).foregroundStyle(Tokens.palette.mutedFg).frame(width: 28)
                        RemoteImage(serverPath: item.posterPath) { Image(systemName: "film").foregroundStyle(Tokens.palette.mutedFg) }
                            .frame(width: 44, height: 66)
                            .background(Tokens.palette.muted)
                            .clipShape(.rect(cornerRadius: 6))
                            .accessibilityHidden(true)
                        VStack(alignment: .leading, spacing: 2) {
                            Text(verbatim: item.title).font(.body.weight(.medium)).lineLimit(2)
                            Text(verbatim: StatsFormat.note(item)).font(.footnote).foregroundStyle(Tokens.palette.mutedFg)
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
