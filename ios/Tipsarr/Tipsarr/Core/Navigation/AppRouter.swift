import SwiftUI

/// Selected tab and the navigation paths deep links can push onto.
@MainActor @Observable
final class AppRouter {
    var tab: AppTab = .discover
    var discoverPath = NavigationPath()
    var requestsPath = NavigationPath()
    var libraryPath = NavigationPath()

    /// Shows the screen of a link. Titles open at the root of their tab with the new screen on top.
    func open(_ target: DeepLinkTarget, api: TipsarrAPI) async {
        switch target {
        case .media(let type, let id):
            discoverPath = NavigationPath()
            discoverPath.append(MediaRoute(type: type, tmdbId: id, title: ""))
            tab = .discover
        case .request(let id):
            guard let record = try? await api.request(id: id) else { return tab = .requests }
            requestsPath = NavigationPath()
            requestsPath.append(record)
            tab = .requests
        case .issue:
            // Issues arrive with the admin screens; Discover is the safe landing until then.
            tab = .discover
        case .library:
            libraryPath = NavigationPath()
            tab = .library
        case .requests:
            requestsPath = NavigationPath()
            tab = .requests
        case .discover:
            discoverPath = NavigationPath()
            tab = .discover
        }
    }
}
