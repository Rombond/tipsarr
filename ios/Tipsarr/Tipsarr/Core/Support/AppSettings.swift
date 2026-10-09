import SwiftUI

enum AppTheme: String, CaseIterable, Sendable {
    case system, light, dark

    var colorScheme: ColorScheme? {
        switch self {
        case .system: nil
        case .light: .light
        case .dark: .dark
        }
    }

    var title: LocalizedStringResource {
        switch self {
        case .system: "theme.system"
        case .light: "theme.light"
        case .dark: "theme.dark"
        }
    }

    var symbol: String {
        switch self {
        case .system: "iphone"
        case .light: "sun.max"
        case .dark: "moon"
        }
    }
}

/// Language the person picked in Settings; nil follows the iPhone.
enum AppLanguage {
    static let supported: [(code: String, name: String)] = [("en", "English"), ("fr", "Français")]
    private static let key = "languageOverride"

    static var override: String? {
        get { UserDefaults.standard.string(forKey: key) }
        set { UserDefaults.standard.set(newValue, forKey: key) }
    }

    /// Bundle that holds the strings of the chosen language (the main bundle when following the iPhone).
    static var bundle: Bundle {
        guard let code = override, let path = Bundle.main.path(forResource: code, ofType: "lproj"), let bundle = Bundle(path: path) else {
            return .main
        }
        return bundle
    }

    static var locale: Locale {
        override.map { Locale(identifier: $0) } ?? .autoupdatingCurrent
    }
}

/// Appearance and language, kept in `UserDefaults`; views read it from the environment.
@MainActor @Observable
final class AppSettings {
    var theme: AppTheme {
        didSet { UserDefaults.standard.set(theme.rawValue, forKey: "theme") }
    }

    var language: String? {
        didSet { AppLanguage.override = language }
    }

    init() {
        theme = AppTheme(rawValue: UserDefaults.standard.string(forKey: "theme") ?? "") ?? .system
        language = AppLanguage.override
    }
}
