import Foundation
import Observation

@MainActor @Observable
final class SearchModel {
    enum Phase: Equatable { case idle, loading, loaded, failed(APIError) }

    private let api: TipsarrAPI
    private(set) var phase: Phase = .idle
    private(set) var items: [MediaItem] = []
    private(set) var people: [PersonSummary] = []
    private(set) var loadingMore = false
    private(set) var movieGenres: [Genre] = []
    private(set) var tvGenres: [Genre] = []
    private(set) var recents: [String]

    private var page = 0
    private var hasMore = false
    private var currentQuery = ""
    private static let recentsKey = "recentSearches"

    init(api: TipsarrAPI) {
        self.api = api
        recents = UserDefaults.standard.stringArray(forKey: Self.recentsKey) ?? []
    }

    // MARK: Genres (landing)

    func loadGenres() async {
        guard movieGenres.isEmpty else { return }
        async let movies = try? api.genres(.movie)
        async let shows = try? api.genres(.tv)
        movieGenres = await movies ?? []
        tvGenres = await shows ?? []
    }

    // MARK: Recents

    func remember(_ query: String) {
        let text = query.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !text.isEmpty else { return }
        recents.removeAll { $0.caseInsensitiveCompare(text) == .orderedSame }
        recents.insert(text, at: 0)
        recents = Array(recents.prefix(8))
        UserDefaults.standard.set(recents, forKey: Self.recentsKey)
    }

    func clearRecents() {
        recents = []
        UserDefaults.standard.removeObject(forKey: Self.recentsKey)
    }

    // MARK: Search

    func search(_ query: String) async {
        let text = query.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !text.isEmpty else {
            phase = .idle
            items = []
            people = []
            return
        }
        currentQuery = text
        if items.isEmpty && people.isEmpty { phase = .loading }
        do {
            let result = try await api.search(text, page: 1)
            guard currentQuery == text else { return }
            items = Self.unique(result.items)
            people = result.people
            page = result.page
            hasMore = result.hasMore
            phase = .loaded
        } catch {
            guard currentQuery == text, !Task.isCancelled else { return }
            phase = .failed(APIError.from(error))
        }
    }

    func loadMore(after item: MediaItem) async {
        guard phase == .loaded, hasMore, !loadingMore,
              let index = items.firstIndex(of: item), index >= items.count - 6 else { return }
        loadingMore = true
        defer { loadingMore = false }
        let text = currentQuery
        do {
            let next = try await api.search(text, page: page + 1)
            guard currentQuery == text else { return }
            let known = Set(items.map(\.id))
            items += Self.unique(next.items).filter { !known.contains($0.id) }
            page = next.page
            hasMore = next.hasMore
        } catch {
            hasMore = false
        }
    }

    private static func unique(_ items: [MediaItem]) -> [MediaItem] {
        var seen = Set<String>()
        return items.filter { seen.insert($0.id).inserted }
    }
}
