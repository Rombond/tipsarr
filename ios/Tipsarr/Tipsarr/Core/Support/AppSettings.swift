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

    var title: LText {
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
        // The language follows the iPhone's per-app language setting now; forget a choice made in an older build.
        AppLanguage.override = nil
        language = nil
    }
}

/// The region the app saved by itself from the iPhone's, per account. While the profile still has that region, a new
/// iPhone region (after a language and region change) replaces it at the next launch. A region the person picked
/// is never replaced: one picked in Settings here is recorded as "" (picked), one picked on the web app or Android
/// differs from the recorded region.
enum AutoRegion {
    private static func key(_ accountID: String) -> String { "autoRegion.\(accountID)" }

    /// nil: nothing recorded yet. "": the person picked the region in Settings. Else the region the app saved.
    static func saved(for accountID: String) -> String? {
        UserDefaults.standard.string(forKey: key(accountID))
    }

    static func record(_ region: String, for accountID: String) {
        UserDefaults.standard.set(region, forKey: key(accountID))
    }

    /// The person picked a region: the app stops following the iPhone's.
    static func markPicked(for accountID: String) { record("", for: accountID) }

    /// The iPhone's region to save on the profile, or nil to keep the profile's.
    static func pending(profileRegion: String, deviceRegion: String?, accountID: String) -> String? {
        guard let device = deviceRegion?.uppercased(), device.count == 2, device.allSatisfy(\.isLetter) else { return nil }
        let current = profileRegion.uppercased()
        if current.isEmpty { return device }
        let saved = saved(for: accountID)
        // Nothing recorded (signed in with an older build): a region equal to the iPhone's is taken as the app's own.
        if saved == nil, current == device {
            record(current, for: accountID)
            return nil
        }
        guard current != device, let saved, saved.uppercased() == current else { return nil }
        return device
    }
}
