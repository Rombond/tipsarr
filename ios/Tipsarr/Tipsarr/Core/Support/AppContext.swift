import SwiftUI

/// What any signed-in screen needs: the API for the active account and who is using it.
struct AppContext: Sendable {
    var api: TipsarrAPI
    var profile: Profile
    /// `userFolderChoice` from `/status`: non-admins may pick the folder when true.
    var userFolderChoice: Bool

    var canChooseFolder: Bool { profile.isAdmin || userFolderChoice }
}

private struct AppContextKey: EnvironmentKey {
    static let defaultValue: AppContext? = nil
}

extension EnvironmentValues {
    var appContext: AppContext? {
        get { self[AppContextKey.self] }
        set { self[AppContextKey.self] = newValue }
    }
}
