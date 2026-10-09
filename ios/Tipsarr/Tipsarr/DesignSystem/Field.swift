import SwiftUI

/// Text input with the app's chrome. Use `secure` for passwords.
struct Field: View {
    let title: LocalizedStringKey
    @Binding var text: String
    var prompt: LocalizedStringKey?
    var secure = false
    var error: LocalizedStringResource?

    @FocusState private var focused: Bool

    var body: some View {
        VStack(alignment: .leading, spacing: Tokens.Spacing.xs) {
            Text(title)
                .font(.footnote.weight(.medium))
                .foregroundStyle(Tokens.palette.mutedFg)
            input
                .focused($focused)
                .padding(.horizontal, Tokens.Spacing.md)
                .frame(minHeight: Tokens.Size.touchTarget)
                .background(Tokens.palette.card, in: .rect(cornerRadius: Tokens.Radius.md))
                .overlay {
                    RoundedRectangle(cornerRadius: Tokens.Radius.md)
                        .strokeBorder(borderColor, lineWidth: focused || error != nil ? 2 : 1)
                }
            if let error {
                Label { Text(error) } icon: { Image(systemName: "exclamationmark.circle") }
                    .font(.footnote)
                    .foregroundStyle(Tokens.palette.destructive)
            }
        }
    }

    @ViewBuilder private var input: some View {
        if secure {
            SecureField(title, text: $text, prompt: prompt.map { Text($0) })
        } else {
            TextField(title, text: $text, prompt: prompt.map { Text($0) })
        }
    }

    private var borderColor: Color {
        if error != nil { Tokens.palette.destructive } else if focused { Tokens.palette.ring } else { Tokens.palette.border }
    }
}

#Preview {
    @Previewable @State var address = ""
    @Previewable @State var password = "secret"
    VStack(spacing: Tokens.Spacing.lg) {
        Field(title: "m.connect.address", text: $address, prompt: "m.connect.placeholder")
        Field(title: "m.connect.address", text: $password, secure: true, error: "m.connect.unreachable")
    }
    .padding()
}
