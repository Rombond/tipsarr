import Foundation

/// Web strings carry plurals as "one | other" (see `seasons.episodes`). This picks the form and fills `%1$@`.
enum Plural {
    static func text(_ key: String, count: Int) -> String {
        let raw = AppLanguage.bundle.localizedString(forKey: key, value: nil, table: nil)
        let forms = raw.components(separatedBy: " | ")
        // English: one only for 1. French: one for 0 and 1.
        let singular = AppLanguage.locale.language.languageCode?.identifier == "fr" ? count <= 1 : count == 1
        let form = forms.count > 1 ? (singular ? forms[0] : forms[1]) : raw
        return form.replacingOccurrences(of: "%1$@", with: String(count)).replacingOccurrences(of: "%@", with: String(count))
    }
}

enum L10n {
    /// Localized string for a dynamic key with `%@` / `%1$@` arguments (SwiftUI cannot do this with a runtime key).
    static func string(_ key: String, _ args: CVarArg...) -> String {
        String(format: AppLanguage.bundle.localizedString(forKey: key, value: nil, table: nil), arguments: args)
    }
}
