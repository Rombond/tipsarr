import Foundation
import Observation

/// Numbers and recent requests shown on the Profile tab.
@MainActor @Observable
final class ProfileModel {
    private let api: TipsarrAPI
    private(set) var requestCount: Int?
    private(set) var watchlistCount: Int?
    private(set) var watchedCount: Int?
    private(set) var recent: [RequestRecord] = []

    init(api: TipsarrAPI) { self.api = api }

    func load() async {
        async let requests = try? api.myRequests(take: 3)
        async let watchlist = try? api.watchlist()
        var watched = LibraryFilters()
        watched.watched = .yes
        async let library = try? api.library(watched, page: 1, pageSize: 1)
        if let page = await requests {
            requestCount = page.total
            recent = page.items
        }
        if let list = await watchlist { watchlistCount = list.count }
        if let page = await library { watchedCount = page.total }
    }
}
