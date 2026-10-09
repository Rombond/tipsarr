import SwiftUI

enum TipsarrButtonKind: Sendable {
    case primary, secondary, ghost, destructive
}

struct TipsarrButtonStyle: ButtonStyle {
    var kind: TipsarrButtonKind = .primary
    var fullWidth = false
    /// Smaller button for use inside list rows (design: 32 pt tall, 14 pt text).
    var compact = false

    @Environment(\.isEnabled) private var isEnabled

    func makeBody(configuration: Configuration) -> some View {
        configuration.label
            .font(compact ? .subheadline.weight(.semibold) : .headline)
            .padding(.horizontal, compact ? Tokens.Spacing.md : Tokens.Spacing.lg)
            .frame(minHeight: compact ? 36 : Tokens.Size.touchTarget)
            .frame(maxWidth: fullWidth ? .infinity : nil)
            .foregroundStyle(foreground)
            .background(background, in: .rect(cornerRadius: Tokens.Radius.md))
            .overlay {
                if kind == .secondary {
                    RoundedRectangle(cornerRadius: Tokens.Radius.md).strokeBorder(Tokens.palette.border)
                }
            }
            .opacity(isEnabled ? (configuration.isPressed ? 0.8 : 1) : 0.4)
            .contentShape(.rect(cornerRadius: Tokens.Radius.md))
            .animation(.easeOut(duration: Tokens.Motion.fast), value: configuration.isPressed)
    }

    private var foreground: Color {
        switch kind {
        case .primary: Tokens.palette.primaryFg
        case .secondary, .ghost: Tokens.palette.fg
        case .destructive: Tokens.palette.bg
        }
    }

    private var background: Color {
        switch kind {
        case .primary: Tokens.palette.primary
        case .secondary: Tokens.palette.card
        case .ghost: .clear
        case .destructive: Tokens.palette.destructive
        }
    }
}

extension ButtonStyle where Self == TipsarrButtonStyle {
    static var tipsarr: TipsarrButtonStyle { .init() }
    static func tipsarr(_ kind: TipsarrButtonKind, fullWidth: Bool = false, compact: Bool = false) -> TipsarrButtonStyle {
        .init(kind: kind, fullWidth: fullWidth, compact: compact)
    }
}

#Preview {
    VStack(spacing: Tokens.Spacing.md) {
        Button("media.request") {}.buttonStyle(.tipsarr(.primary, fullWidth: true))
        Button("common.cancel") {}.buttonStyle(.tipsarr(.secondary, fullWidth: true))
        Button("common.retry") {}.buttonStyle(.tipsarr(.ghost))
        Button("common.delete") {}.buttonStyle(.tipsarr(.destructive, fullWidth: true))
        Button("common.save") {}.buttonStyle(.tipsarr(.primary, fullWidth: true)).disabled(true)
    }
    .padding()
}
