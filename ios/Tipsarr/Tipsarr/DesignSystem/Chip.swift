import SwiftUI

/// Selectable filter chip (Discover genres, Requests filters).
struct Chip: View {
    let title: LocalizedStringResource
    var isSelected = false
    var action: () -> Void = {}

    var body: some View {
        Button(action: action) {
            Text(title)
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
