import SwiftUI

/// What any signed-in screen needs: the API for the active account and who is using it.
struct AppContext: Sendable {
    var api: TipsarrAPI
    var profile: Profile
    /// Choosing the folder of a request is the advanced-requests permission.
    var canChooseFolder: Bool { profile.can(.requestAdvanced) }
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
