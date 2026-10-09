import SwiftUI

/// Selectable filter chip (Discover genres, Requests filters).
struct Chip: View {
    let title: LText
    var isSelected = false
    /// Small count shown after the title (0 or nil hides it).
    var count: Int?
    var action: () -> Void = {}

    var body: some View {
        Button(action: action) {
            HStack(spacing: Tokens.Spacing.xs) {
                Text(title)
                if let count, count > 0 {
                    Text(verbatim: "\(count)").font(.caption.weight(.semibold)).opacity(0.7)
                }
            }
                .font(.subheadline.weight(.medium))
                .padding(.horizontal, Tokens.Spacing.md)
                .frame(minHeight: Tokens.Size.touchTarget - Tokens.Spacing.sm)
                .foregroundStyle(isSelected ? Tokens.palette.primaryFg : Tokens.palette.fg)
                .background(isSelected ? Tokens.palette.primary : Tokens.palette.muted, in: .capsule)
                .contentShape(.capsule)
        }
        .buttonStyle(.plain)
        .accessibilityAddTraits(isSelected ? .isSelected : [])
        .animation(.easeOut(duration: Tokens.Motion.fast), value: isSelected)
    }
}

#Preview {
    HStack {
        Chip(title: "m.tab.discover", isSelected: true)
        Chip(title: "m.tab.library")
        Chip(title: "m.tab.requests")
    }
    .padding()
}
