import SwiftUI

struct StatusBadge: View {
    let state: RequestState
    var compact = false

    var body: some View {
        Label {
            if !compact { Text(state.title) }
        } icon: {
            Image(systemName: state.symbol)
        }
        .font(.caption.weight(.semibold))
        .foregroundStyle(state.color)
        .padding(.horizontal, compact ? Tokens.Spacing.xs : Tokens.Spacing.sm)
        .padding(.vertical, Tokens.Spacing.xs)
        .background(state.color.opacity(0.15), in: .capsule)
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
