import SwiftUI

@main
struct TipsarrApp: App {
    @UIApplicationDelegateAdaptor(AppDelegate.self) private var appDelegate
    @State private var settings = AppSettings()

    init() {
        #if DEBUG
        // Line-buffered output, so `[network]` / `[fold]` lines reach a piped console at once.
        setvbuf(stdout, nil, _IOLBF, 0)
        #endif
    }

    var body: some Scene {
        WindowGroup {
            Root()
                .environment(settings)
                .environment(\.locale, settings.language.map { Locale(identifier: $0) } ?? .autoupdatingCurrent)
                .preferredColorScheme(settings.theme.colorScheme)
        }
    }
}

/// `-catalogue` (debug builds) opens the component catalogue instead of the app.
private struct Root: View {
    var body: some View {
        #if DEBUG
        if ProcessInfo.processInfo.arguments.contains("-catalogue") { Catalogue() } else { RootView() }
        #else
        RootView()
        #endif
    }
}
