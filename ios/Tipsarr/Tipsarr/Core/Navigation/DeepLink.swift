import Foundation

enum DeepLinkTarget: Equatable, Sendable {
    case media(MediaType, Int)
    case request(String)
    case issue(String)
    case library
    case requests
    case discover
}

/// `tipsarr://<server-host>/media/{movie|tv}/{tmdbId}`, `/request/{id}`, `/issue/{id}`, `/library`, `/requests`.
/// Unknown paths open Discover.
struct DeepLink: Equatable, Sendable {
    var host: String
    var target: DeepLinkTarget

    static func parse(_ url: URL) -> DeepLink? {
        guard url.scheme?.lowercased() == "tipsarr", let host = url.host(), !host.isEmpty else { return nil }
        let parts = url.pathComponents.filter { $0 != "/" }
        let target: DeepLinkTarget
        switch (parts.first, parts.count) {
        case ("media", 3):
            if let type = MediaType(rawValue: parts[1]), let id = Int(parts[2]) { target = .media(type, id) } else { target = .discover }
        case ("request", 2): target = .request(parts[1])
        case ("issue", 2): target = .issue(parts[1])
        case ("library", _): target = .library
        case ("requests", _): target = .requests
        default: target = .discover
        }
        return DeepLink(host: host.lowercased(), target: target)
    }
}

extension DeepLink {
    /// Link that opens a title in the app of someone signed in to the same server.
    static func mediaURL(server: URL, type: MediaType, tmdbId: Int) -> URL? {
        guard let host = server.host() else { return nil }
        return URL(string: "tipsarr://\(host)/media/\(type.rawValue)/\(tmdbId)")
    }
}
