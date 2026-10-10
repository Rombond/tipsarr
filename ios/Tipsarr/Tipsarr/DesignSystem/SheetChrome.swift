import SwiftUI

extension View {
    /// Shared sheet look: token corner radius, grabber, card background.
    /// `detents` nil: the content sets its own (a sheet that fits its content).
    func tipsarrSheet(detents: Set<PresentationDetent>? = [.medium, .large]) -> some View {
        self
            // A centred form sheet on iPad and the Duo inner display; bottom sheet with detents on iPhone.
            .presentationSizing(.form)
            .modifier(OptionalDetents(detents: detents))
            .presentationDragIndicator(.visible)
            .presentationCornerRadius(Tokens.Radius.sheet)
            .presentationBackground(Tokens.palette.bg)
    }
}

extension View {
    /// Keeps a list or form a readable width on iPad and the Duo inner display, centred in the screen.
    func readableColumn(_ width: CGFloat = 720) -> some View {
        frame(maxWidth: width).frame(maxWidth: .infinity)
    }
}

private struct OptionalDetents: ViewModifier {
    let detents: Set<PresentationDetent>?

    @ViewBuilder func body(content: Content) -> some View {
        if let detents { content.presentationDetents(detents) } else { content }
    }
}
