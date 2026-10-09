import Foundation

/// One signed-in user on one server. The token lives in the Keychain with the rest.
struct Account: Codable, Identifiable, Hashable, Sendable {
    var id: String { Self.makeID(server: serverURL, userID: userID) }
    var serverURL: URL
    var userID: String
    var name: String
    var token: String
    var lastUsed: Date

    static func makeID(server: URL, userID: String) -> String {
        "\(server.host() ?? server.absoluteString)|\(userID)"
    }
}
