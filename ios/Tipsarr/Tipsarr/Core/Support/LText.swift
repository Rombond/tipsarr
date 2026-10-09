import SwiftUI

/// Text that is either a catalog key or an already-final string.
/// `LocalizedStringResource` always uses the iPhone's language; `Text(LocalizedStringKey)` follows the
/// `\.locale` environment, so this type goes through the latter and the in-app language switch works.
struct LText: ExpressibleByStringLiteral, Sendable {
    private enum Storage: Sendable {
        case key(String)
        case verbatim(String)
    }

    private let storage: Storage

    init(stringLiteral value: String) { storage = .key(value) }

    /// A string that must be shown as is (server messages, names, numbers).
    static func verbatim(_ text: String) -> LText {
        LText(storage: .verbatim(text))
    }

    private init(storage: Storage) { self.storage = storage }

    /// The text in the chosen language, for places that need a `String`.
    var resolved: String {
        switch storage {
        case .key(let key): AppLanguage.bundle.localizedString(forKey: key, value: nil, table: nil)
        case .verbatim(let text): text
        }
    }

    fileprivate var view: Text {
        switch storage {
        case .key(let key): Text(LocalizedStringKey(key))
        case .verbatim(let text): Text(verbatim: text)
        }
    }
}

extension Text {
    init(_ text: LText) { self = text.view }
}

/// Picks one of two texts. Needed because `Text(flag ? "a" : "b")` infers `String` and skips localization.
func choose(_ condition: Bool, _ yes: LText, _ no: LText) -> LText { condition ? yes : no }
