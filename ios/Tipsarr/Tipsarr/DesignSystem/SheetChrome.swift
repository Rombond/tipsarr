import SwiftUI

extension View {
    /// Shared sheet look: token corner radius, grabber, card background.
    func tipsarrSheet(detents: Set<PresentationDetent> = [.medium, .large]) -> some View {
        self
            // A centred form sheet on iPad and the Duo inner display; bottom sheet with detents on iPhone.
            .presentationSizing(.form)
            .presentationDetents(detents)
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
