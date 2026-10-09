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
    @State private var width: CGFloat = 0
    @Environment(\.liveUpdates) private var live
    @Binding var path: NavigationPath
    var openSearch: () -> Void = {}

    init(api: TipsarrAPI, path: Binding<NavigationPath>, openSearch: @escaping () -> Void = {}) {
        _model = State(initialValue: DiscoverModel(api: api))
        _path = path
        self.openSearch = openSearch
    }

    private var wide: Bool { width >= 900 }

    var body: some View {
        NavigationStack(path: $path) {
            ScrollView {
                VStack(alignment: .leading, spacing: Tokens.Spacing.lg) {
                    // Landscape iPad home: a hero and rails (Penpot page 10) instead of chips on top.
                    if wide && chip == .forYou {
                        WideHome(model: model) { chips }
                    } else {
                        chips
                    }
                    switch chip.source {
                    case nil: ForYouContent(home: model.home)
                    case let source?: MediaGrid(model: model.list(source))
                    }
                }
                .padding(.bottom, Tokens.Spacing.sm)
            }
            .onGeometryChange(for: CGFloat.self, of: { $0.size.width }) { width = $0 }
            .refreshable { await refreshCurrent() }
            .onChange(of: live?.suggestionsTick) { if chip == .forYou { Task { await model.home.refresh() } } }
            .onChange(of: live?.requestsTick) { Task { await refreshCurrent() } }
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

struct Rail: View {
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
                        MediaLink(route: item.route) {
                            PosterCard(item: item)
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
                        MediaLink(route: item.route) {
                            PosterCard(item: item)
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

/// Landscape iPad "For you": a hero on the top trending title, then the chips and the suggestion rows.
/// The Trending and Upcoming rails are not repeated: they are chips of their own.
private struct WideHome<Chips: View>: View {
    let model: DiscoverModel
    @ViewBuilder var chips: Chips

    var body: some View {
        let trending = model.list(.trending)
        VStack(alignment: .leading, spacing: Tokens.Spacing._2xl) {
            if let first = trending.items.first { Hero(item: first) }
            chips
        }
        .task { await trending.loadIfNeeded() }
    }
}

/// Big banner of one title: backdrop, title, short facts, overview and a button to its page.
private struct Hero: View {
    let item: MediaItem

    var body: some View {
        RemoteImage(path: item.backdropPath ?? item.posterPath, size: .w1280) { Tokens.palette.muted }
            .frame(height: 380)
            .frame(maxWidth: .infinity)
            .overlay {
                LinearGradient(colors: [.black.opacity(0.1), .black.opacity(0.75)], startPoint: .top, endPoint: .bottom)
            }
            .overlay(alignment: .bottomLeading) {
                VStack(alignment: .leading, spacing: Tokens.Spacing.md) {
                    Text(verbatim: item.title).font(.system(size: 40, weight: .bold)).lineLimit(2).accessibilityAddTraits(.isHeader)
                    Text(verbatim: [item.releaseYear, L10n.string(item.type == .tv ? "type.tv" : "type.movie")].compactMap { $0 }.joined(separator: " · "))
                        .font(.subheadline.weight(.medium)).opacity(0.85)
                    if let overview = item.overview {
                        Text(verbatim: overview).font(.body).opacity(0.9).lineLimit(3).frame(maxWidth: 560, alignment: .leading)
                    }
                    HStack(spacing: Tokens.Spacing.md) {
                        MediaLink(route: item.route) {
                            Label { Text("media.view_details") } icon: { Image(systemName: "info.circle") }
                        }
                        .buttonStyle(.tipsarr(.primary))
                        if let state = item.state { StatusBadge(state: state) }
                    }
                }
                .foregroundStyle(.white)
                .padding(Tokens.Spacing._3xl)
            }
            .clipShape(.rect(cornerRadius: Tokens.Radius.xl))
            .padding(.horizontal, Tokens.Spacing.lg)
            .accessibilityElement(children: .contain)
    }
}
