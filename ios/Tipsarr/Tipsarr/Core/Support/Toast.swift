import SwiftUI

/// One short message at a time, shown above the content for a few seconds.
@MainActor @Observable
final class ToastCenter {
    struct Message: Equatable, Identifiable {
        enum Kind { case success, error }
        let id = UUID()
        var text: String
        var kind: Kind
    }

    private(set) var message: Message?
    private var dismissTask: Task<Void, Never>?

    func show(_ text: String, kind: Message.Kind = .success) {
        let message = Message(text: text, kind: kind)
        self.message = message
        dismissTask?.cancel()
        dismissTask = Task {
            try? await Task.sleep(for: .seconds(3))
            if !Task.isCancelled, self.message == message { self.message = nil }
        }
    }
}

struct ToastOverlay: ViewModifier {
    let center: ToastCenter

    func body(content: Content) -> some View {
        content
            .overlay(alignment: .top) {
                if let message = center.message {
                    Label {
                        Text(verbatim: message.text).font(.subheadline.weight(.medium))
                    } icon: {
                        Image(systemName: message.kind == .success ? "checkmark.circle.fill" : "exclamationmark.triangle.fill")
                            .foregroundStyle(message.kind == .success ? Tokens.Status.available : Tokens.palette.destructive)
                    }
                    .padding(.horizontal, Tokens.Spacing.lg)
                    .padding(.vertical, Tokens.Spacing.md)
                    .background(Tokens.palette.card, in: .capsule)
                    .overlay { Capsule().strokeBorder(Tokens.palette.border) }
                    .shadow(color: .black.opacity(0.15), radius: 12, y: 4)
                    .padding(.top, Tokens.Spacing.sm)
                    .transition(.move(edge: .top).combined(with: .opacity))
                    .accessibilityAddTraits(.isStaticText)
                    .onAppear { UIAccessibility.post(notification: .announcement, argument: message.text) }
                }
            }
            .animation(.easeOut(duration: Tokens.Motion.normal), value: center.message)
            .sensoryFeedback(trigger: center.message) { _, new in
                new.map { $0.kind == .success ? .success : .error }
            }
    }
}
