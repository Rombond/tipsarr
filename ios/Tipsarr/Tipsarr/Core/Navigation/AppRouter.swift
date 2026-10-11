import SwiftUI

/// Selected tab and the navigation paths deep links can push onto.
@MainActor @Observable
final class AppRouter {
    var tab: AppTab = {
        #if DEBUG
        // `-startTab requests` on the command line: opens that tab (screenshots from the console).
        switch UserDefaults.standard.string(forKey: "startTab") {
        case "requests": return .requests
        case "library": return .library
        case "search": return .search
        default: break
        }
        #endif
        return .discover
    }()
    var discoverPath = NavigationPath()
    var requestsPath = NavigationPath()
    var libraryPath = NavigationPath()
    var profilePath = NavigationPath()
    /// Profile (with Settings and the admin tools) is a sheet over the tabs.
    var profileOpen: Bool = {
        #if DEBUG
        return UserDefaults.standard.string(forKey: "startTab") == "profile"
        #else
        return false
        #endif
    }()
    /// Selected request in the two-column Requests screen (wide windows); the stack uses `requestsPath`.
    var requestsSelection: String?

    /// Settings is its own sheet, separate from Profile.
    var settingsOpen = false
    var settingsPath = NavigationPath()

    func openProfile() {
        profilePath = NavigationPath()
        profileOpen = true
    }

    func openSettings() {
        settingsPath = NavigationPath()
        settingsOpen = true
    }

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
        case .issue(let id):
            if let pane {
                pane.content = .profile(.issue(id))
            } else {
                profilePath = NavigationPath()
                profilePath.append(ProfileRoute.issue(id))
                profileOpen = true
            }
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
