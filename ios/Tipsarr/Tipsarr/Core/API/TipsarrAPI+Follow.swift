import Foundation
import OpenAPIRuntime

/// What the server keeps for a followed request (the same state the pushes carry).
struct RequestFollowState: Sendable, Equatable {
    var requestId: String
    var stage: String
    var percent: Int
    var etaSeconds: Int
    var updatedAt: Int
    var expiresAt: Int

    var contentState: RequestActivityAttributes.ContentState {
        .init(stage: stage, percent: percent, etaSeconds: etaSeconds, updatedAt: updatedAt)
    }
}

extension RequestFollowState {
    init(_ follow: Components.Schemas.RequestFollow) {
        self.init(requestId: follow.requestId, stage: follow.stage.rawValue, percent: Int(follow.percent),
                  etaSeconds: Int(follow.etaSeconds), updatedAt: Int(follow.updatedAt), expiresAt: Int(follow.expiresAt))
    }
}

extension TipsarrAPI {
    /// Idempotent: called again whenever the activity push token changes.
    func followRequest(id: String, pushToken: String, activityId: String?, sandbox: Bool, server: String) async throws -> RequestFollowState {
        try await load { client in
            let body = Components.Schemas.FollowRequestInputBody(activityId: activityId, pushToken: pushToken, sandbox: sandbox, server: server)
            guard case .ok(let ok) = try await client.followRequest(path: .init(id: id), body: .json(body)) else { throw APIError.unexpected }
            return RequestFollowState(try ok.body.json)
        }
    }

    func unfollowRequest(id: String) async throws {
        try await load { client in _ = try await client.unfollowRequest(path: .init(id: id)) }
    }

    /// Follows this session has on the server.
    func listFollows() async throws -> [RequestFollowState] {
        try await load { client in
            guard case .ok(let ok) = try await client.listFollows() else { throw APIError.unexpected }
            return try ok.body.json.items.map(RequestFollowState.init)
        }
    }
}
