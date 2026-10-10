import SwiftUI

/// Lifecycle of a request or title, shared by badges, timelines and posters.
enum RequestState: String, CaseIterable, Sendable {
    case requested, approved, searching, downloading, available, partial, declined, failed

    var title: LText {
        switch self {
        case .requested: "state.requested"
        case .approved: "state.approved"
        case .searching: "state.searching"
        case .downloading: "state.downloading"
        case .available: "state.available"
        case .partial: "state.partial"
        case .declined: "state.declined"
        case .failed: "state.failed"
        }
    }

    var color: Color {
        switch self {
        case .requested: Tokens.Status.requested
        case .approved: Tokens.Status.approved
        case .searching: Tokens.Status.searching
        case .downloading: Tokens.Status.downloading
        case .available: Tokens.Status.available
        case .partial: Tokens.Status.partial
        case .declined: Tokens.Status.declined
        case .failed: Tokens.Status.failed
        }
    }

    /// Same symbols without the circle: on a solid badge a circle glyph would look hollow.
    var solidSymbol: String {
        switch self {
        case .approved, .available: "checkmark"
        case .downloading: "arrow.down"
        case .declined: "xmark"
        default: symbol
        }
    }

    /// Colour is never the only signal: each state has its own symbol.
    var symbol: String {
        switch self {
        case .requested: "clock"
        case .approved: "checkmark.circle"
        case .searching: "magnifyingglass"
        case .downloading: "arrow.down.circle"
        case .available: "checkmark.circle.fill"
        case .partial: "circle.lefthalf.filled"
        case .declined: "xmark.circle"
        case .failed: "exclamationmark.triangle"
        }
    }
}
