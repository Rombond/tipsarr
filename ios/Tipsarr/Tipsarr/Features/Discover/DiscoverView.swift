import SwiftUI

enum DiscoverChip: CaseIterable, Hashable {
    case trending, upcoming, movies, tv

    var title: LocalizedStringResource {
        switch self {
        case .trending: "discover.trending"
        case .upcoming: "m.discover.chip_upcoming"
        case .movies: "m.discover.chip_movies"
        case .tv: "m.discover.chip_tv"
        }
    }

    var source: MediaListModel.Source? {
        switch self {
        case .trending: nil
        case .upcoming: .upcoming
        case .movies: .movies
        case .tv: .tv
        }
    }
}

enum DiscoverRoute: Hashable { case allTrending }

struct DiscoverView: View {
    @State private var model: DiscoverModel
    @State private var chip: DiscoverChip = .trending
    var openSearch: () -> Void = {}

    init(api: TipsarrAPI, openSearch: @escaping () -> Void = {}) {
        _model = State(initialValue: DiscoverModel(api: api))
        self.openSearch = openSearch
    }

    var body: some View {
        NavigationStack {
            ScrollView {
                LazyVStack(alignment: .leading, spacing: Tokens.Spacing.lg) {
                    chips
                    switch chip.source {
                    case nil: TrendingContent(home: model.home)
                    case let source?: MediaGrid(model: model.list(source))
                    }
                }
                .padding(.vertical, Tokens.Spacing.sm)
            }
            .background(Tokens.palette.bg)
            .navigationTitle("m.tab.discover")
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button(action: openSearch) { Image(systemName: "magnifyingglass") }
                        .accessibilityLabel(Text("m.discover.search"))
                }
            }
            .navigationDestination(for: MediaRoute.self) { MediaRoutePlaceholder(route: $0) }
            .navigationDestination(for: DiscoverRoute.self) { _ in
                MediaGridScreen(title: "discover.trending", model: model.list(.trending))
            }
        }
    }

    private var chips: some View {
        ScrollView(.horizontal, showsIndicators: false) {
            HStack(spacing: Tokens.Spacing.sm) {
                ForEach(DiscoverChip.allCases, id: \.self) { item in
                    Chip(title: item.title, isSelected: chip == item) { chip = item }
                }
            }
            .padding(.horizontal, Tokens.Spacing.lg)
        }
    }
}

/// Trending chip: rails.
private struct TrendingContent: View {
    let home: DiscoverHomeModel

    var body: some View {
        Group {
            switch home.phase {
            case .idle, .loading:
                RailSkeleton()
            case .failed(let error):
                ErrorState(error: error) { await home.refresh() }
            case .loaded:
                VStack(alignment: .leading, spacing: Tokens.Spacing._2xl) {
                    Rail(title: "m.discover.trending_week", items: home.trending, seeAll: DiscoverRoute.allTrending)
                    ForEach(home.rows) { row in
                        Rail(title: LocalizedStringResource(stringLiteral: row.title), items: row.items)
                    }
                }
            }
        }
        .task { await home.loadIfNeeded() }
    }
}

private struct Rail<Route: Hashable>: View {
    let title: LocalizedStringResource
    let items: [MediaItem]
    var seeAll: Route?

    var body: some View {
        VStack(alignment: .leading, spacing: Tokens.Spacing.md) {
            HStack {
                Text(title).font(.title3.weight(.semibold)).accessibilityAddTraits(.isHeader)
                Spacer()
                if let seeAll {
                    NavigationLink(value: seeAll) { Text("m.discover.see_all").font(.subheadline) }
                        .foregroundStyle(Tokens.palette.mutedFg)
                }
            }
            .padding(.horizontal, Tokens.Spacing.lg)
            ScrollView(.horizontal, showsIndicators: false) {
                LazyHStack(alignment: .top, spacing: Tokens.Spacing.md) {
                    ForEach(items) { item in
                        NavigationLink(value: item.route) {
                            PosterCard(title: item.title, subtitle: item.releaseYear, posterPath: item.posterPath,
                                       posterSize: .w342, state: item.state)
                                .frame(width: 120)
                        }
                        .buttonStyle(.plain)
                    }
                }
                .padding(.horizontal, Tokens.Spacing.lg)
            }
        }
    }
}

extension Rail where Route == Never? {
    init(title: LocalizedStringResource, items: [MediaItem]) {
        self.init(title: title, items: items, seeAll: nil)
    }
}

private struct RailSkeleton: View {
    var body: some View {
        VStack(alignment: .leading, spacing: Tokens.Spacing._2xl) {
            ForEach(0..<2, id: \.self) { _ in
                VStack(alignment: .leading, spacing: Tokens.Spacing.md) {
                    SkeletonBlock(height: 22).frame(width: 180).skeleton()
                    HStack(spacing: Tokens.Spacing.md) {
                        ForEach(0..<3, id: \.self) { _ in PosterSkeleton().frame(width: 120) }
                    }
                }
                .padding(.horizontal, Tokens.Spacing.lg)
            }
        }
    }
}

/// Non-trending chips and "See all": adaptive grid with infinite scroll.
struct MediaGrid: View {
    let model: MediaListModel

    var body: some View {
        Group {
            switch model.phase {
            case .idle, .loading:
                LazyVGrid(columns: Self.columns, spacing: Tokens.Spacing.lg) {
                    ForEach(0..<9, id: \.self) { _ in PosterSkeleton() }
                }
                .padding(.horizontal, Tokens.Spacing.lg)
            case .failed(let error):
                ErrorState(error: error) { await model.refresh() }
            case .loaded:
                LazyVGrid(columns: Self.columns, spacing: Tokens.Spacing.lg) {
                    ForEach(model.items) { item in
                        NavigationLink(value: item.route) {
                            PosterCard(title: item.title, subtitle: item.releaseYear, posterPath: item.posterPath, state: item.state)
                        }
                        .buttonStyle(.plain)
                        .task { await model.loadMore(after: item) }
                    }
                }
                .padding(.horizontal, Tokens.Spacing.lg)
                if model.loadingMore { ProgressView().frame(maxWidth: .infinity).padding() }
            }
        }
        .task { await model.loadIfNeeded() }
    }

    private static let columns = [GridItem(.adaptive(minimum: 104, maximum: 180), spacing: Tokens.Spacing.md, alignment: .top)]
}

struct MediaGridScreen: View {
    let title: LocalizedStringResource
    let model: MediaListModel

    var body: some View {
        ScrollView { MediaGrid(model: model).padding(.vertical, Tokens.Spacing.sm) }
            .background(Tokens.palette.bg)
            .navigationTitle(Text(title))
            .navigationBarTitleDisplayMode(.inline)
            .refreshable { await model.refresh() }
    }
}

/// Error view for a failed list: offline gets the offline wording, others the server message.
struct ErrorState: View {
    let error: APIError
    let retry: () async -> Void

    var body: some View {
        Group {
            if error == .unreachable {
                StateView.offline { Task { await retry() } }
            } else {
                StateView(symbol: "exclamationmark.triangle", title: LocalizedStringResource(stringLiteral: error.localizedMessage),
                          actionTitle: "common.retry") { Task { await retry() } }
            }
        }
        .frame(minHeight: 360)
    }
}

/// Replaced by the real detail screen in step 4.
struct MediaRoutePlaceholder: View {
    let route: MediaRoute

    var body: some View {
        Text(verbatim: route.title)
            .font(.title.weight(.bold))
            .padding()
            .frame(maxWidth: .infinity, maxHeight: .infinity)
            .background(Tokens.palette.bg)
            .navigationTitle(Text(verbatim: route.title))
            .navigationBarTitleDisplayMode(.inline)
    }
}
