import SwiftUI

extension View {
    /// Shared sheet look: token corner radius, grabber, card background.
    func tipsarrSheet(detents: Set<PresentationDetent> = [.medium, .large]) -> some View {
        self
            .presentationDetents(detents)
            .presentationDragIndicator(.visible)
            .presentationCornerRadius(Tokens.Radius.sheet)
            .presentationBackground(Tokens.palette.bg)
    }
}
