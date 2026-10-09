import Foundation
import Observation

@MainActor @Observable
final class RequestsModel {
    enum Phase: Equatable { case idle, loading, loaded, failed(APIError) }

    private let api: TipsarrAPI
    let isAdmin: Bool
    private(set) var phase: Phase = .idle
    private(set) var items: [RequestRecord] = []
    private(set) var counts = RequestCounts()
    private(set) var busy: Set<String> = []
    private(set) var loadingMore = false
    private var total = 0
    private var currentFilter: RequestFilter = .all

    init(api: TipsarrAPI, isAdmin: Bool) {
        self.api = api
        self.isAdmin = isAdmin
    }

    // MARK: Loading

    func load(_ filter: RequestFilter) async {
        currentFilter = filter
        if phase == .idle || phase == .loading { phase = .loading }
        async let freshCounts = try? api.requestCounts()
        do {
            let page = try await api.requests(filter, skip: 0)
            guard currentFilter == filter else { return }
            items = page.items
            total = page.total
            phase = .loaded
        } catch {
            guard currentFilter == filter, !Task.isCancelled else { return }
            if items.isEmpty { phase = .failed(APIError.from(error)) }
        }
        if let freshCounts = await freshCounts { counts = freshCounts }
    }

    func loadMore(after record: RequestRecord) async {
        guard phase == .loaded, !loadingMore, items.count < total,
              let index = items.firstIndex(of: record), index >= items.count - 4 else { return }
        loadingMore = true
        defer { loadingMore = false }
        let filter = currentFilter
        if let page = try? await api.requests(filter, skip: items.count), currentFilter == filter {
            let known = Set(items.map(\.id))
            items += page.items.filter { !known.contains($0.id) }
            total = page.total
        }
    }

    // MARK: Actions (return the updated record; the list follows)

    func approve(_ record: RequestRecord) async throws -> RequestRecord {
        try await act(record) { try await self.api.approve(id: record.id) }
    }

    func decline(_ record: RequestRecord, reason: String) async throws -> RequestRecord {
        try await act(record) { try await self.api.decline(id: record.id, reason: reason) }
    }

    func retry(_ record: RequestRecord) async throws -> RequestRecord {
        try await act(record) { try await self.api.retryRequest(id: record.id) }
    }

    func delete(_ record: RequestRecord) async throws {
        busy.insert(record.id)
        defer { busy.remove(record.id) }
        try await api.deleteRequest(id: record.id)
        items.removeAll { $0.id == record.id }
        total = max(0, total - 1)
        await refreshCounts()
    }

    private func act(_ record: RequestRecord, _ work: () async throws -> RequestRecord) async throws -> RequestRecord {
        busy.insert(record.id)
        defer { busy.remove(record.id) }
        let updated = try await work()
        if let index = items.firstIndex(where: { $0.id == record.id }) {
            if Self.matches(updated, currentFilter) { items[index] = updated } else { items.remove(at: index) }
        }
        await refreshCounts()
        return updated
    }

    private func refreshCounts() async {
        if let fresh = try? await api.requestCounts() { counts = fresh }
    }

    private static func matches(_ record: RequestRecord, _ filter: RequestFilter) -> Bool {
        switch filter {
        case .all: true
        case .pending: record.state == .requested
        case .approved: [.approved, .searching, .downloading].contains(record.state)
        case .available: record.state == .available
        case .declined: record.state == .declined
        case .failed: record.state == .failed
        }
    }

    // MARK: Permissions

    /// Admins can delete anything; owners only unfinished requests (the server enforces it too).
    func canDelete(_ record: RequestRecord) -> Bool {
        isAdmin || record.isOpen
    }
}
