import Foundation

/// Lean copies of what the app needs from the generated schemas, so screens
/// never depend on generated types.
struct ServerStatus: Sendable, Equatable {
    var version: String
    var apiVersion: Int
    var minAppVersion: String
    var userFolderChoice: Bool
    var defaultLanguage: String
    var pushAvailable: Bool
}

struct Profile: Sendable, Equatable, Codable {
    var id: String
    var name: String
    var role: String
    var region: String
    var language: String
    var ratingSource: String

    var isAdmin: Bool { role == "admin" }
}

struct SignInResult: Sendable {
    var token: String
    var profile: Profile
}

/// A server the user connected to, with the status read at that moment.
struct Server: Sendable, Equatable {
    var url: URL
    var status: ServerStatus

    var host: String { url.host() ?? url.absoluteString }
}
