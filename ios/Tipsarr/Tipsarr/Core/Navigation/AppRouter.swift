import SwiftUI

/// Selected tab and the navigation paths deep links can push onto.
@MainActor @Observable
final class AppRouter {
    var tab: AppTab = .discover
    var discoverPath = NavigationPath()
    var requestsPath = NavigationPath()
    var libraryPath = NavigationPath()
    /// Selected request in the two-column Requests screen (wide windows); the stack uses `requestsPath`.
    var requestsSelection: String?

    /// Shows the screen of a link. Titles open at the root of their tab with the new screen on top.
    func open(_ target: DeepLinkTarget, api: TipsarrAPI, pane: DetailPane? = nil) async {
        switch target {
        case .media(let type, let id):
            let route = MediaRoute(type: type, tmdbId: id, title: "")
            if let pane {
                // Unfolded Duo: the title opens in the right pane.
                pane.content = .media(route)
                tab = .discover
            } else {
                discoverPath = NavigationPath()
                discoverPath.append(route)
                tab = .discover
            }
        case .request(let id):
            guard let record = try? await api.request(id: id) else { return tab = .requests }
            requestsPath = NavigationPath()
            requestsPath.append(record)
            requestsSelection = record.id
            pane?.content = .request(record)
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
