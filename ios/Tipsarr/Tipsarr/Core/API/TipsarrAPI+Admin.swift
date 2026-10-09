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
