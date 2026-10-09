import SwiftUI

struct StatusBadge: View {
    let state: RequestState
    var compact = false
    /// Pill with a tinted background (default) or just the coloured icon and word, with no padding.
    var plain = false

    var body: some View {
        Label {
            if !compact { Text(state.title) }
        } icon: {
            Image(systemName: state.symbol)
        }
        .font(.caption.weight(.semibold))
        .foregroundStyle(state.color)
        .padding(.horizontal, plain ? 0 : (compact ? Tokens.Spacing.xs : Tokens.Spacing.sm))
        .padding(.vertical, plain ? 0 : Tokens.Spacing.xs)
        .background(plain ? Color.clear : state.color.opacity(0.15), in: .capsule)
        .accessibilityElement(children: .ignore)
        .accessibilityLabel(Text(state.title))
    }
}

#Preview {
    VStack(alignment: .leading, spacing: Tokens.Spacing.sm) {
        ForEach(RequestState.allCases, id: \.self) { StatusBadge(state: $0) }
        HStack { ForEach(RequestState.allCases, id: \.self) { StatusBadge(state: $0, compact: true) } }
    }
    .padding()
}
