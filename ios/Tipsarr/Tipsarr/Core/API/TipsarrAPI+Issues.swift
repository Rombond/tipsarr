import Foundation
import OpenAPIRuntime

extension TipsarrAPI {
    func issues(_ filter: IssueFilter, skip: Int, take: Int = 20) async throws -> IssuePage {
        try await load { client in
            guard case .ok(let ok) = try await client.listIssues(
                query: .init(filter: .init(rawValue: filter.rawValue), take: Int64(take), skip: Int64(skip))
            ) else { throw APIError.unexpected }
            let body = try ok.body.json
            return IssuePage(items: body.items.map(IssueRecord.init), total: Int(body.total))
        }
    }

    func openIssueCount() async throws -> Int {
        try await load { client in
            guard case .ok(let ok) = try await client.issueCounts() else { throw APIError.unexpected }
            return Int(try ok.body.json.open)
        }
    }

    func issue(id: String) async throws -> IssueThread {
        try await load { client in
            guard case .ok(let ok) = try await client.getIssue(path: .init(id: id)) else { throw APIError.unexpected }
            return IssueThread(try ok.body.json)
        }
    }

    func comment(on id: String, message: String) async throws -> IssueThread {
        try await load { client in
            guard case .ok(let ok) = try await client.commentIssue(path: .init(id: id), body: .json(.init(message: message))) else { throw APIError.unexpected }
            return IssueThread(try ok.body.json)
        }
    }

    func resolveIssue(id: String) async throws -> IssueThread {
        try await load { client in
            guard case .ok(let ok) = try await client.resolveIssue(path: .init(id: id)) else { throw APIError.unexpected }
            return IssueThread(try ok.body.json)
        }
    }

    func reopenIssue(id: String) async throws -> IssueThread {
        try await load { client in
            guard case .ok(let ok) = try await client.reopenIssue(path: .init(id: id)) else { throw APIError.unexpected }
            return IssueThread(try ok.body.json)
        }
    }

    func deleteIssue(id: String) async throws {
        try await load { client in _ = try await client.deleteIssue(path: .init(id: id)) }
    }
}

extension IssueRecord {
    init(_ v: Components.Schemas.IssueView) {
        self.init(
            id: v.id, type: v._type == .tv ? .tv : .movie, tmdbId: Int(v.tmdbId), title: v.title, posterPath: v.posterPath,
            kind: IssueKind(rawValue: v.kind.rawValue) ?? .other, status: v.status == .resolved ? .resolved : .open,
            season: v.season.map(Int.init), episode: v.episode.map(Int.init),
            createdBy: v.createdBy.name, createdAt: Date(timeIntervalSince1970: TimeInterval(v.createdAt)),
            resolvedBy: v.resolvedBy?.name, commentCount: Int(v.commentCount)
        )
    }
}

extension IssueThread {
    init(_ t: Components.Schemas.IssueThread) {
        self.init(
            issue: IssueRecord(
                id: t.id, type: t._type == .tv ? .tv : .movie, tmdbId: Int(t.tmdbId), title: t.title, posterPath: t.posterPath,
                kind: IssueKind(rawValue: t.kind.rawValue) ?? .other, status: t.status == .resolved ? .resolved : .open,
                season: t.season.map(Int.init), episode: t.episode.map(Int.init),
                createdBy: t.createdBy.name, createdAt: Date(timeIntervalSince1970: TimeInterval(t.createdAt)),
                resolvedBy: t.resolvedBy?.name, commentCount: Int(t.commentCount)
            ),
            comments: t.comments.map {
                IssueComment(id: $0.id, message: $0.message, userID: $0.user.id, userName: $0.user.name,
                             createdAt: Date(timeIntervalSince1970: TimeInterval($0.createdAt)))
            }
        )
    }
}
