import SwiftUI

extension View {
    /// Shared sheet look: token corner radius, grabber, card background.
    /// `detents` nil: the content sets its own (a sheet that fits its content).
    /// `sizing`: `.page` for a sheet that needs room for two columns on iPad (Settings); same bottom sheet on iPhone.
    func tipsarrSheet(detents: Set<PresentationDetent>? = [.medium, .large], sizing: some PresentationSizing = .form) -> some View {
        self
            // A centred form sheet on iPad and the Duo inner display; bottom sheet with detents on iPhone.
            .presentationSizing(sizing)
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

extension View {
    /// A bottom sheet as tall as its content (`contentHeight`, measured by the sheet), up to `maxHeight`; drag up for
    /// the full height. iOS 26 adds the bottom safe area to a custom height on a floating sheet (empty space under the
    /// last button), so it is taken off here.
    func fittedSheetDetents(_ contentHeight: CGFloat, maxHeight: CGFloat = 640) -> some View {
        let height = contentHeight > 0 ? max(contentHeight - Self.windowBottomInset, 1) : 0
        return presentationDetents([.height(min(height, maxHeight)), .large])
    }

    @MainActor private static var windowBottomInset: CGFloat {
        UIApplication.shared.connectedScenes
            .compactMap { $0 as? UIWindowScene }
            .flatMap(\.windows)
            .first(where: \.isKeyWindow)?.safeAreaInsets.bottom ?? 0
    }
}

private struct OptionalDetents: ViewModifier {
    let detents: Set<PresentationDetent>?

    @ViewBuilder func body(content: Content) -> some View {
        if let detents { content.presentationDetents(detents) } else { content }
    }
}
