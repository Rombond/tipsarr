import SwiftUI

/// Full-area empty / error / offline state.
struct StateView: View {
    let symbol: String
    let title: LText
    var message: LText?
    var actionTitle: LText?
    var action: (() -> Void)?

    var body: some View {
        ContentUnavailableView {
            Label { Text(title) } icon: { Image(systemName: symbol) }
        } description: {
            if let message { Text(message) }
        } actions: {
            if let actionTitle, let action {
                Button(action: action) { Text(actionTitle) }
                    .buttonStyle(.tipsarr(.primary))
            }
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }
}

extension StateView {
    static func offline(retry: @escaping () -> Void) -> StateView {
        StateView(symbol: "wifi.slash", title: "m.offline.title", message: "m.offline.body",
                  actionTitle: "m.offline.retry", action: retry)
    }

    static func noResults() -> StateView {
        StateView(symbol: "magnifyingglass", title: "common.no_results")
    }
}

#Preview("Offline") { StateView.offline {} }
#Preview("No results") { StateView.noResults() }
