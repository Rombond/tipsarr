import SwiftUI

struct Banner: View {
    enum Kind: Sendable {
        case info, warning, error

        var color: Color {
            switch self {
            case .info: Tokens.Status.approved
            case .warning: Tokens.Status.requested
            case .error: Tokens.palette.destructive
            }
        }

        var symbol: String {
            switch self {
            case .info: "info.circle"
            case .warning: "exclamationmark.triangle"
            case .error: "xmark.octagon"
            }
        }
    }

    var kind: Kind = .info
    let title: LText
    var message: LText?
    var onDismiss: (() -> Void)?

    var body: some View {
        HStack(alignment: .top, spacing: Tokens.Spacing.md) {
            Image(systemName: kind.symbol)
                .foregroundStyle(kind.color)
                .accessibilityHidden(true)
            VStack(alignment: .leading, spacing: Tokens.Spacing.xs) {
                Text(title).font(.subheadline.weight(.semibold))
                if let message {
                    Text(message).font(.footnote).foregroundStyle(Tokens.palette.mutedFg)
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            if let onDismiss {
                Button(action: onDismiss) {
                    Image(systemName: "xmark")
                        .frame(minWidth: Tokens.Size.touchTarget, minHeight: Tokens.Size.touchTarget)
                        .contentShape(.rect)
                }
                .buttonStyle(.plain)
                .padding(-Tokens.Spacing.md)
                .accessibilityLabel(Text("common.dismiss"))
            }
        }
        .padding(Tokens.Spacing.md)
        .background(kind.color.opacity(0.12), in: .rect(cornerRadius: Tokens.Radius.md))
        .overlay { RoundedRectangle(cornerRadius: Tokens.Radius.md).strokeBorder(kind.color.opacity(0.4)) }
        .accessibilityElement(children: .combine)
    }
}

#Preview {
    VStack(spacing: Tokens.Spacing.md) {
        Banner(title: "banner.dry_run", message: "banner.dry_run_long")
        Banner(kind: .warning, title: "m.update.title", message: "m.update.body")
        Banner(kind: .error, title: "m.offline.title", message: "m.offline.body", onDismiss: {})
    }
    .padding()
}
