import Foundation
import Observation

@MainActor @Observable
final class IssuesModel {
    enum Phase: Equatable { case idle, loading, loaded, failed(APIError) }

    private let api: TipsarrAPI
    private(set) var items: [IssueRecord] = []
    private(set) var phase: Phase = .idle
    private(set) var openCount = 0
    private(set) var loadingMore = false
    private var total = 0
    private var current: IssueFilter = .open

    init(api: TipsarrAPI) { self.api = api }

    func load(_ filter: IssueFilter) async {
        current = filter
        if items.isEmpty { phase = .loading }
        async let count = try? api.openIssueCount()
        do {
            let page = try await api.issues(filter, skip: 0)
            guard current == filter else { return }
            items = page.items
            total = page.total
            phase = .loaded
        } catch {
            guard current == filter, !Task.isCancelled else { return }
            if items.isEmpty { phase = .failed(APIError.from(error)) }
        }
        if let count = await count { openCount = count }
    }

    func loadMore(after item: IssueRecord) async {
        guard phase == .loaded, !loadingMore, items.count < total,
              let index = items.firstIndex(of: item), index >= items.count - 4 else { return }
        loadingMore = true
        defer { loadingMore = false }
        let filter = current
        if let page = try? await api.issues(filter, skip: items.count), current == filter {
            let known = Set(items.map(\.id))
            items += page.items.filter { !known.contains($0.id) }
            total = page.total
        }
    }

    func remove(_ id: String) {
        items.removeAll { $0.id == id }
        total = max(0, total - 1)
    }
}
