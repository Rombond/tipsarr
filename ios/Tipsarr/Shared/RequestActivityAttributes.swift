import ActivityKit
import Foundation

/// A followed request shown as a Live Activity. Shared by the app and the widget extension.
/// `ContentState` must match the `content-state` the push relay sends (keys and types).
struct RequestActivityAttributes: ActivityAttributes {
    struct ContentState: Codable, Hashable, Sendable {
        /// requested | approved | searching | downloading | available | declined | failed
        var stage: String
        /// 0–100; live while downloading, 100 when available, else 0.
        var percent: Int
        /// 0 = unknown.
        var etaSeconds: Int
        /// Unix seconds the state was computed.
        var updatedAt: Int
    }

    var requestId: String
    /// Local `Account.id`.
    var accountId: String
    /// `host` or `host:port` (PushManager.serverKey); the tap opens `tipsarr://<server>/request/<id>`.
    var server: String
    var title: String
    /// movie | tv
    var mediaType: String
    /// Stage to its label, resolved by the app (the extension does no localization lookup).
    var labels: [String: String]
}

extension RequestActivityAttributes {
    static let finalStages: Set<String> = ["available", "declined", "failed"]

    var deepLink: URL? { URL(string: "tipsarr://\(server)/request/\(requestId)") }
}

/// Posters shared through the App Group, so the extension can show them without network.
enum LivePoster {
    static let appGroup = "group.com.brebond.tipsarr-app"

    static func url(requestId: String) -> URL? {
        FileManager.default.containerURL(forSecurityApplicationGroupIdentifier: appGroup)?
            .appending(path: "live", directoryHint: .isDirectory)
            .appending(path: "\(requestId).jpg")
    }
}
