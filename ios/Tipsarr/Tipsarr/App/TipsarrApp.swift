import SwiftUI

@main
struct TipsarrApp: App {
    @State private var settings = AppSettings()

    var body: some Scene {
        WindowGroup {
            RootView()
                .environment(settings)
                .environment(\.locale, settings.language.map { Locale(identifier: $0) } ?? .autoupdatingCurrent)
                .preferredColorScheme(settings.theme.colorScheme)
        }
    }
}
