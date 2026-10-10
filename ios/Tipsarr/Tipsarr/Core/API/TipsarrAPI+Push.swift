import Foundation
import OpenAPIRuntime

/// What the server keeps for this install (never the token itself).
struct PushRegistration: Sendable, Equatable {
    var categories: Int
    var language: String
    var sandbox: Bool
}

extension TipsarrAPI {
    /// `server` is the host this app reaches the server on; it comes back in every alert. `categories` nil keeps what the server has (a refresh at launch).
    func registerDevice(token: String, sandbox: Bool, language: String, server: String, categories: Int?) async throws -> PushRegistration {
        try await load { client in
            let body = Components.Schemas.RegisterDeviceInputBody(
                categories: categories.map { Int64($0) },
                language: language,
                pushToken: token,
                sandbox: sandbox,
                server: server
            )
            guard case .ok(let ok) = try await client.registerDevice(body: .json(body)) else { throw APIError.unexpected }
            let device = try ok.body.json
            return PushRegistration(categories: Int(device.categories), language: device.language, sandbox: device.sandbox)
        }
    }

    func unregisterDevice() async throws {
        try await load { client in _ = try await client.unregisterDevice() }
    }
}
