import Foundation
import OpenAPIRuntime

extension TipsarrAPI {
    func users() async throws -> [Profile] {
        try await load { client in
            guard case .ok(let ok) = try await client.listUsers() else { throw APIError.unexpected }
            return try ok.body.json.map(Profile.init)
        }
    }

    func userDetail(id: String) async throws -> UserDetail {
        try await load { client in
            guard case .ok(let ok) = try await client.userProfile(path: .init(id: id)) else { throw APIError.unexpected }
            let v = try ok.body.json
            let s = v.stats
            return UserDetail(
                profile: Profile(id: v.id, name: v.name, role: v.role.rawValue, region: v.region, language: v.language,
                                 ratingSource: v.ratingSource, createdAt: Date(timeIntervalSince1970: TimeInterval(v.createdAt)),
                                 lastLoginAt: Date(timeIntervalSince1970: TimeInterval(v.lastLoginAt))),
                stats: UserStats(requests: Int(s.requests), movies: Int(s.movies), shows: Int(s.shows), pending: Int(s.pending),
                                 approved: Int(s.approved), available: Int(s.available), declined: Int(s.declined),
                                 failed: Int(s.failed), watchlist: Int(s.watchlist), watched: Int(s.watched))
            )
        }
    }

    func setRole(id: String, admin: Bool) async throws -> Profile {
        try await load { client in
            guard case .ok(let ok) = try await client.updateUser(path: .init(id: id), body: .json(.init(role: admin ? .admin : .user))) else { throw APIError.unexpected }
            return Profile(try ok.body.json)
        }
    }

    func syncStatus() async throws -> SyncStatus {
        try await load { client in
            guard case .ok(let ok) = try await client.syncStatus() else { throw APIError.unexpected }
            let body = try ok.body.json
            return SyncStatus(
                canSync: body.canSync,
                jobs: body.jobs.map { job in
                    JobStatus(name: job.name, running: job.running, status: job.status, message: job.message,
                              lastStarted: job.lastStartedAt > 0 ? Date(timeIntervalSince1970: TimeInterval(job.lastStartedAt)) : nil,
                              lastFinished: job.lastFinishedAt > 0 ? Date(timeIntervalSince1970: TimeInterval(job.lastFinishedAt)) : nil,
                              everySeconds: Int(job.everySeconds))
                },
                movies: Int(body.movies), shows: Int(body.shows)
            )
        }
    }

    func runJob(_ name: String) async throws {
        try await load { client in
            guard let job = Operations.runSyncJob.Input.Path.jobPayload(rawValue: name) else { throw APIError.unexpected }
            _ = try await client.runSyncJob(path: .init(job: job))
        }
    }
}

extension TipsarrAPI {
    /// `user`: nil is the signed-in person, "all" everyone (admins), or a user id (admins).
    func stats(period: StatsPeriod, user: String?) async throws -> StatsReport {
        try await load { client in
            guard case .ok(let ok) = try await client.getStats(query: .init(user: user, period: .init(rawValue: period.rawValue))) else { throw APIError.unexpected }
            let r = try ok.body.json
            func top(_ list: [Components.Schemas.StatsTop]) -> [StatsTop] {
                list.map {
                    StatsTop(type: $0._type == .tv ? .tv : .movie, tmdbId: Int($0.tmdbId), title: $0.title, year: $0.year.map(Int.init),
                             plays: Int($0.plays), hours: $0.hours, posterPath: $0.posterUrl.flatMap { $0.isEmpty ? nil : $0 })
                }
            }
            return StatsReport(
                hours: r.totals.hours, plays: Int(r.totals.plays), titles: Int(r.totals.titles), movies: Int(r.totals.movies), shows: Int(r.totals.shows),
                requestsMade: Int(r.requests.made), requestsAvailable: Int(r.requests.available), requestsDeclined: Int(r.requests.declined),
                genres: r.genres.map { StatsBucket(name: $0.name, hours: $0.hours, titles: Int($0.titles)) },
                decades: r.decades.map { StatsBucket(name: $0.name, hours: $0.hours, titles: Int($0.titles)) },
                months: r.months.map { StatsMonth(month: $0.month, hours: $0.hours, plays: Int($0.plays)) },
                weekdays: r.weekdays, hoursOfDay: r.hoursOfDay,
                top: top(r.top), topMovies: top(r.topMovies), topShows: top(r.topShows),
                exact: r.source == .plugin, pluginHint: r.plugin.hint
            )
        }
    }
}

extension TipsarrAPI {
    /// Available requests whose requester has not watched them. `user` nil = everyone.
    func unwatchedRequests(user: String?) async throws -> RequestPage {
        try await load { client in
            guard case .ok(let ok) = try await client.listRequests(query: .init(filter: .unwatched, user: user, take: 40)) else { throw APIError.unexpected }
            let list = try ok.body.json
            return RequestPage(items: list.items.map(RequestRecord.init), total: Int(list.total))
        }
    }

    /// Library titles nobody ever played, oldest first (admins).
    func neverWatched() async throws -> LibraryPage {
        try await load { client in
            let query = Operations.listLibrary.Input.Query(neverWatched: true, sort: .added, dir: .asc, page: 1, pageSize: 30)
            guard case .ok(let ok) = try await client.listLibrary(query: query) else { throw APIError.unexpected }
            let body = try ok.body.json
            return LibraryPage(items: body.items.map(LibraryItem.init), page: Int(body.page), totalPages: Int(body.totalPages), total: Int(body.total))
        }
    }
}
