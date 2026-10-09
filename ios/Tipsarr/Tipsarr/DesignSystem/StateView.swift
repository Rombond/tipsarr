import SwiftUI

/// Full-area empty / error / offline state.
struct StateView: View {
    let symbol: String
    let title: LText
    var message: LText?
    var actionTitle: LText?
    var action: (() -> Void)?

    var body: some View {
        VStack(spacing: Tokens.Spacing.lg) {
            Image(systemName: symbol)
                .font(.largeTitle)
                .foregroundStyle(Tokens.palette.mutedFg)
                .accessibilityHidden(true)
            VStack(spacing: Tokens.Spacing.sm) {
                Text(title).font(.title3.weight(.semibold))
                if let message {
                    Text(message).font(.subheadline).foregroundStyle(Tokens.palette.mutedFg)
                }
            }
            .multilineTextAlignment(.center)
            if let actionTitle, let action {
                Button(action: action) { Text(actionTitle) }
                    .buttonStyle(.tipsarr(.primary))
            }
        }
        .padding(Tokens.Spacing._3xl)
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .accessibilityElement(children: .contain)
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
