import SwiftUI

struct StatusBadge: View {
    let state: RequestState
    var compact = false
    /// Colour fill with a white icon, like the web badge: stays readable over a poster.
    var solid = false

    var body: some View {
        // Not `Label`: its icon-to-text gap is too wide for a small pill.
        HStack(spacing: Tokens.Spacing.xs) {
            Image(systemName: solid ? state.solidSymbol : state.symbol)
            if !compact { Text(state.title) }
        }
        .font(solid && compact ? .caption2.weight(.bold) : .caption.weight(.semibold))
        .foregroundStyle(solid ? Color.white : state.color)
        .padding(.horizontal, compact ? Tokens.Spacing.xs : Tokens.Spacing.sm)
        .padding(.vertical, Tokens.Spacing.xs)
        // A solid compact badge is a circle, whatever the symbol's width.
        .frame(minWidth: solid && compact ? 24 : nil, minHeight: solid && compact ? 24 : nil)
        .background(solid ? state.color : state.color.opacity(0.15), in: .capsule)
        .accessibilityElement(children: .ignore)
        .accessibilityLabel(Text(state.title))
    }
}

/// Small neutral "4K" pill next to a request's status badge, when its quality profile allows 2160p.
struct UHDBadge: View {
    var body: some View {
        Text("req.uhd_badge")
            .font(.caption.weight(.semibold))
            .foregroundStyle(Tokens.palette.mutedFg)
            .padding(.horizontal, Tokens.Spacing.sm)
            .padding(.vertical, Tokens.Spacing.xs)
            .background(Tokens.palette.muted, in: .capsule)
    }
}

/// The status badge, with the 4K pill after it when the request is 4K.
struct RequestBadges: View {
    let record: RequestRecord

    var body: some View {
        HStack(spacing: Tokens.Spacing.xs) {
            StatusBadge(state: record.state)
            if record.uhd { UHDBadge() }
        }
    }
}

#Preview {
    VStack(alignment: .leading, spacing: Tokens.Spacing.sm) {
        ForEach(RequestState.allCases, id: \.self) { StatusBadge(state: $0) }
        HStack { ForEach(RequestState.allCases, id: \.self) { StatusBadge(state: $0, compact: true) } }
    }
    .padding()
}
