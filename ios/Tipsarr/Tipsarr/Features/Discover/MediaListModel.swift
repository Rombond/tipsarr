import Foundation
import Observation

/// One paged list of titles (infinite scroll). Duplicates across pages are dropped.
@MainActor @Observable
final class MediaListModel {
    enum Source: Sendable { case trending, upcoming, movies, tv }
    enum Phase: Equatable { case idle, loading, loaded, failed(APIError) }

    let source: Source
    private let api: TipsarrAPI
    private(set) var items: [MediaItem] = []
    private(set) var phase: Phase = .idle
    private(set) var loadingMore = false
    private var page = 0
    private var hasMore = true

    init(source: Source, api: TipsarrAPI) {
        self.source = source
        self.api = api
    }

    func loadIfNeeded() async {
        if phase == .idle { await refresh() }
    }

    func refresh() async {
        if items.isEmpty { phase = .loading }
        do {
            let first = try await fetch(page: 1)
            items = Self.unique(first.items)
            page = first.page
            hasMore = first.hasMore
            phase = .loaded
        } catch {
            // Keep what is on screen when a pull to refresh fails.
            if items.isEmpty { phase = .failed(APIError.from(error)) }
        }
    }

    /// Call from the last rows; loads the next page once.
    func loadMore(after item: MediaItem) async {
        guard phase == .loaded, hasMore, !loadingMore,
              let index = items.firstIndex(of: item), index >= items.count - 6 else { return }
        loadingMore = true
        defer { loadingMore = false }
        do {
            let next = try await fetch(page: page + 1)
            let known = Set(items.map(\.id))
            items += Self.unique(next.items).filter { !known.contains($0.id) }
            page = next.page
            hasMore = next.hasMore
        } catch {
            hasMore = false
        }
    }

    private func fetch(page: Int) async throws -> MediaPage {
        switch source {
        case .trending: try await api.trending(page: page)
        case .upcoming: try await api.upcoming(page: page)
        case .movies: try await api.popularMovies(page: page)
        case .tv: try await api.popularTV(page: page)
        }
    }

    private static func unique(_ items: [MediaItem]) -> [MediaItem] {
        var seen = Set<String>()
        return items.filter { seen.insert($0.id).inserted }
    }
}

/// The Trending chip: a trending rail plus the suggestion rows.
@MainActor @Observable
final class DiscoverHomeModel {
    enum Phase: Equatable { case idle, loading, loaded, failed(APIError) }

    private let api: TipsarrAPI
    private(set) var trending: [MediaItem] = []
    private(set) var rows: [SuggestionRow] = []
    private(set) var phase: Phase = .idle

    init(api: TipsarrAPI) { self.api = api }

    func loadIfNeeded() async {
        if phase == .idle { await refresh() }
    }

    func refresh() async {
        if trending.isEmpty { phase = .loading }
        async let suggestions = try? api.suggestions()
        do {
            trending = Array(try await api.trending(page: 1).items.prefix(15))
            rows = await suggestions?.filter { !$0.items.isEmpty } ?? rows
            phase = .loaded
        } catch {
            if trending.isEmpty { phase = .failed(APIError.from(error)) }
        }
    }
}

/// Everything the Discover tab owns for one account.
@MainActor @Observable
final class DiscoverModel {
    let home: DiscoverHomeModel
    private let api: TipsarrAPI
    private var lists: [MediaListModel.Source: MediaListModel] = [:]

    init(api: TipsarrAPI) {
        self.api = api
        home = DiscoverHomeModel(api: api)
    }

    func list(_ source: MediaListModel.Source) -> MediaListModel {
        if let model = lists[source] { return model }
        let model = MediaListModel(source: source, api: api)
        lists[source] = model
        return model
    }
}
