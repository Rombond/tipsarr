import SwiftUI

enum TipsarrButtonKind: Sendable {
    case primary, secondary, tonal, ghost, destructive
}

struct TipsarrButtonStyle: ButtonStyle {
    var kind: TipsarrButtonKind = .primary
    var fullWidth = false
    /// Smaller button for use inside list rows (design: 32 pt tall, 14 pt text).
    var compact = false
    /// Fully rounded ends, to sit next to round buttons (detail action row).
    var capsule = false

    @Environment(\.isEnabled) private var isEnabled

    func makeBody(configuration: Configuration) -> some View {
        configuration.label
            .font(compact ? .subheadline.weight(.semibold) : .headline)
            .padding(.horizontal, compact ? Tokens.Spacing.md : Tokens.Spacing.lg)
            .frame(minHeight: compact ? 36 : Tokens.Size.touchTarget)
            .frame(maxWidth: fullWidth ? .infinity : nil)
            .foregroundStyle(foreground)
            .background(background, in: shape)
            .overlay {
                if kind == .secondary { shape.stroke(Tokens.palette.border) }
            }
            .opacity(isEnabled ? (configuration.isPressed ? 0.8 : 1) : 0.4)
            .contentShape(shape)
            .animation(.easeOut(duration: Tokens.Motion.fast), value: configuration.isPressed)
    }

    private var shape: AnyShape { capsule ? AnyShape(.capsule) : AnyShape(.rect(cornerRadius: Tokens.Radius.md)) }

    private var foreground: Color {
        switch kind {
        case .primary: Tokens.palette.primaryFg
        case .secondary, .tonal, .ghost: Tokens.palette.fg
        case .destructive: Tokens.palette.bg
        }
    }

    private var background: Color {
        switch kind {
        case .primary: Tokens.palette.primary
        case .secondary: Tokens.palette.card
        case .tonal: Tokens.palette.muted
        case .ghost: .clear
        case .destructive: Tokens.palette.destructive
        }
    }
}

extension ButtonStyle where Self == TipsarrButtonStyle {
    static var tipsarr: TipsarrButtonStyle { .init() }
    static func tipsarr(_ kind: TipsarrButtonKind, fullWidth: Bool = false, compact: Bool = false, capsule: Bool = false) -> TipsarrButtonStyle {
        .init(kind: kind, fullWidth: fullWidth, compact: compact, capsule: capsule)
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
