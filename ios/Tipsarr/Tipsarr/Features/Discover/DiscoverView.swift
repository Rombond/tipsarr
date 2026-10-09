import SwiftUI

enum DiscoverChip: CaseIterable, Hashable {
    case forYou, trending, upcoming, movies, tv

    var title: LText {
        switch self {
        case .forYou: "m.discover.chip_for_you"
        case .trending: "discover.trending"
        case .upcoming: "m.discover.chip_upcoming"
        case .movies: "m.discover.chip_movies"
        case .tv: "m.discover.chip_tv"
        }
    }

    var source: MediaListModel.Source? {
        switch self {
        case .forYou: nil
        case .trending: .trending
        case .upcoming: .upcoming
        case .movies: .movies
        case .tv: .tv
        }
    }
}

struct DiscoverView: View {
    @State private var model: DiscoverModel
    @State private var chip: DiscoverChip = .forYou
    var openSearch: () -> Void = {}

    init(api: TipsarrAPI, openSearch: @escaping () -> Void = {}) {
        _model = State(initialValue: DiscoverModel(api: api))
        self.openSearch = openSearch
    }

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(alignment: .leading, spacing: Tokens.Spacing.lg) {
                    chips
                    switch chip.source {
                    case nil: ForYouContent(home: model.home)
                    case let source?: MediaGrid(model: model.list(source))
                    }
                }
                .padding(.vertical, Tokens.Spacing.sm)
            }
            .refreshable { await refreshCurrent() }
            .background(Tokens.palette.bg)
            .navigationTitle("m.tab.discover")
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button(action: openSearch) { Image(systemName: "magnifyingglass") }
                        .accessibilityLabel(Text("m.discover.search"))
                }
            }
            .mediaDestinations()
        }
    }

    private func refreshCurrent() async {
        if let source = chip.source {
            await model.list(source).refresh()
        } else {
            await model.home.refresh()
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

/// "For you" chip: the suggestion rows (the server decides which rows, e.g. trending, because you watched).
private struct ForYouContent: View {
    let home: DiscoverHomeModel

    var body: some View {
        Group {
            switch home.phase {
            case .idle, .loading:
                RailSkeleton()
            case .failed(let error):
                ErrorState(error: error) { await home.refresh() }
            case .loaded where home.rows.isEmpty:
                StateView(symbol: home.generating ? "hourglass" : "sparkles",
                          title: home.generating ? "suggest.preparing" : "common.no_results")
                    .frame(minHeight: 360)
            case .loaded:
                VStack(alignment: .leading, spacing: Tokens.Spacing._2xl) {
                    ForEach(home.rows) { row in
                        Rail(title: LText.verbatim(row.title), items: row.items)
                    }
                }
            }
        }
        .task { await home.loadIfNeeded() }
    }
}

private struct Rail: View {
    let title: LText
    let items: [MediaItem]

    var body: some View {
        VStack(alignment: .leading, spacing: Tokens.Spacing.md) {
            Text(title)
                .font(.title3.weight(.semibold))
                .accessibilityAddTraits(.isHeader)
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

/// Grid chips (Trending, Upcoming, Movies, TV): adaptive grid with infinite scroll.
/// Must not sit in a `LazyVStack`: nested lazy containers build every cell, which would fire every "load more".
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
        // The same grid view is reused when the chip changes, so the task must follow the model.
        .task(id: ObjectIdentifier(model)) { await model.loadIfNeeded() }
    }

    private static let columns = [GridItem(.adaptive(minimum: 104, maximum: 180), spacing: Tokens.Spacing.md, alignment: .top)]
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
                StateView(symbol: "exclamationmark.triangle", title: LText.verbatim(error.localizedMessage),
                          actionTitle: "common.retry") { Task { await retry() } }
            }
        }
        .frame(minHeight: 360)
    }
}
