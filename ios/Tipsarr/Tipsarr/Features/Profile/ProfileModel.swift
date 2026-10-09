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
    private(set) var openIssues: Int?
    var isAdmin = false
    /// Everything this person watched, most time first (movies and shows mixed).
    private(set) var topWatched: [StatsTop] = []

    init(api: TipsarrAPI) { self.api = api }

    func load() async {
        async let requests = try? api.myRequests(take: 3)
        async let watchlist = try? api.watchlist()
        var watched = LibraryFilters()
        watched.watched = .yes
        async let library = try? api.library(watched, page: 1, pageSize: 1)
        async let stats = try? api.stats(period: .all, user: nil)
        async let issues = isAdmin ? try? api.openIssueCount() : nil
        if let page = await requests {
            requestCount = page.total
            recent = page.items
        }
        if let list = await watchlist { watchlistCount = list.count }
        if let page = await library { watchedCount = page.total }
        if let stats = await stats { topWatched = stats.top }
        if let issues = await issues { openIssues = issues }
    }
}
