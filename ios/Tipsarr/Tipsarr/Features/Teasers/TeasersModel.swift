import Foundation
import Observation

/// The Teasers feed: loads batches ahead of the person, tells the server which titles they swiped past.
@MainActor @Observable
final class TeasersModel {
    enum Phase: Equatable { case loading, loaded, failed(APIError) }

    private let api: TipsarrAPI
    private(set) var items: [Teaser] = []
    private(set) var phase: Phase = .loading
    /// Nothing new is left to show (the end card is the last page).
    private(set) var done = false
    /// Trending titles shown (no video) while the first trailers still download on the server. The server's own default feed is
    /// its pool of trending, upcoming and popular titles plus the person's suggestions, but only titles whose file is ready.
    private(set) var placeholders: [MediaItem] = []
    private var loadingMore = false
    private var reported: Set<String> = []

    /// Cards left before the end when the next batch is fetched.
    private static let prefetchDistance = 3

    init(api: TipsarrAPI) { self.api = api }

    func loadIfNeeded() async {
        guard items.isEmpty, !done else { return }
        await load()
    }

    func load() async {
        phase = .loading
        if items.isEmpty, placeholders.isEmpty { Task { await loadPlaceholders() } }
        do {
            let feed = try await api.teasers(exclude: [])
            items = feed.items
            done = feed.done
            phase = .loaded
        } catch {
            phase = .failed(APIError.from(error))
        }
    }

    private func loadPlaceholders() async {
        guard let page = try? await api.trending(page: 1), items.isEmpty else { return }
        placeholders = Array(page.items.filter { $0.backdropPath != nil || $0.posterPath != nil }.prefix(10))
    }

    /// The card with this id is on screen: the ones before it are seen, and the next batch is fetched when the end is near.
    func visible(_ id: String?) async {
        guard let id, let index = items.firstIndex(where: { $0.id == id }) else { return }
        for teaser in items[..<index] { report(teaser) }
        if index >= items.count - Self.prefetchDistance { await loadMore() }
    }

    /// The end card is on screen: every card was swiped past.
    func reachedEnd() { for teaser in items { report(teaser) } }

    /// Fetches the next batch. Returns the id of the first new card, nil when nothing came (trailers can still be downloading).
    @discardableResult
    func loadMore() async -> String? {
        guard !done, !loadingMore else { return nil }
        loadingMore = true
        defer { loadingMore = false }
        guard let feed = try? await api.teasers(exclude: items.map { "\($0.item.type.rawValue):\($0.item.tmdbId)" }) else { return nil }
        let known = Set(items.map(\.id))
        let fresh = feed.items.filter { !known.contains($0.id) }
        items += fresh
        if feed.done { done = true }
        return fresh.first?.id
    }

    private func report(_ teaser: Teaser) {
        guard reported.insert(teaser.id).inserted else { return }
        let api = api
        Task { try? await api.markTeaserSeen(teaser.item.type, id: teaser.item.tmdbId) }
    }

    /// Drops a card (not interested, or a trailer file that cannot be played) and reports it as seen.
    func remove(_ id: String) {
        guard let teaser = items.first(where: { $0.id == id }) else { return }
        report(teaser)
        items.removeAll { $0.id == id }
    }

    func setState(_ state: RequestState?, for id: String) {
        guard let index = items.firstIndex(where: { $0.id == id }) else { return }
        items[index].item.state = state
    }

    func toggleWatchlist(_ id: String) async throws {
        guard let index = items.firstIndex(where: { $0.id == id }) else { return }
        let on = !items[index].watchlisted
        try await api.setWatchlisted(on, items[index].item.type, id: items[index].item.tmdbId)
        if let now = items.firstIndex(where: { $0.id == id }) { items[now].watchlisted = on }
    }

    func hide(_ id: String) async throws {
        guard let teaser = items.first(where: { $0.id == id }) else { return }
        try await api.setBlocklisted(true, teaser.item.type, id: teaser.item.tmdbId)
        remove(id)
    }
}

extension TeasersModel.Phase {
    var isFailed: Bool { if case .failed = self { true } else { false } }
}
