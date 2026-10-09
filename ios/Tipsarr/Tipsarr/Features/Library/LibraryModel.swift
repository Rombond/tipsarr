import Foundation
import Observation

@MainActor @Observable
final class LibraryModel {
    enum Phase: Equatable { case idle, loading, loaded, failed(APIError) }

    private let api: TipsarrAPI
    var filters = LibraryFilters()
    private(set) var phase: Phase = .idle
    private(set) var items: [LibraryItem] = []
    private(set) var total = 0
    private(set) var facets: LibraryFacets?
    private(set) var loadingMore = false
    private var page = 0
    private var hasMore = false

    init(api: TipsarrAPI) { self.api = api }

    /// True when nothing is filtered and the library has no title: it was never synced.
    var neverSynced: Bool {
        phase == .loaded && items.isEmpty && filters == LibraryFilters()
    }

    func loadFacets() async {
        if facets == nil { facets = try? await api.libraryFacets() }
    }

    func reload() async {
        if items.isEmpty { phase = .loading }
        let current = filters
        do {
            let result = try await api.library(current, page: 1)
            guard current == filters else { return }
            items = result.items
            total = result.total
            page = result.page
            hasMore = result.hasMore
            phase = .loaded
        } catch {
            guard current == filters, !Task.isCancelled else { return }
            if items.isEmpty { phase = .failed(APIError.from(error)) }
        }
    }

    func loadMore(after item: LibraryItem) async {
        guard phase == .loaded, hasMore, !loadingMore,
              let index = items.firstIndex(of: item), index >= items.count - 9 else { return }
        loadingMore = true
        defer { loadingMore = false }
        let current = filters
        if let next = try? await api.library(current, page: page + 1), current == filters {
            let known = Set(items.map(\.id))
            items += next.items.filter { !known.contains($0.id) }
            page = next.page
            hasMore = next.hasMore
        }
    }
}
