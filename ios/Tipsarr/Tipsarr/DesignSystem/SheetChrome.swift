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
    /// the full height. `pinnedFooter`: the content ends with a `safeAreaInset` bar that already stays clear of the
    /// sheet's bottom safe area, so none of it is taken off.
    func fittedSheetDetents(_ contentHeight: CGFloat, maxHeight: CGFloat = 640, pinnedFooter: Bool = false) -> some View {
        modifier(FittedSheetDetents(contentHeight: contentHeight, maxHeight: maxHeight, pinnedFooter: pinnedFooter))
    }
}

/// iOS 26 adds the sheet's bottom safe area to a custom height on a floating iPhone sheet (empty space under the last
/// button of a scrolling form), so it is taken off. The inset is the sheet's own, not the window's: an iPad sheet floats
/// above the home indicator without one and gets exactly the height asked for (taking the window's inset off cut its
/// last line).
private struct FittedSheetDetents: ViewModifier {
    let contentHeight: CGFloat
    let maxHeight: CGFloat
    let pinnedFooter: Bool
    @State private var bottomInset: CGFloat = 0

    func body(content: Content) -> some View {
        let height = contentHeight > 0 ? max(contentHeight - (pinnedFooter ? 0 : bottomInset), 1) : 0
        content
            .onGeometryChange(for: CGFloat.self, of: { $0.safeAreaInsets.bottom }) { bottomInset = $0 }
            .presentationDetents([.height(min(height, maxHeight)), .large])
    }
}

private struct OptionalDetents: ViewModifier {
    let detents: Set<PresentationDetent>?

    @ViewBuilder func body(content: Content) -> some View {
        if let detents { content.presentationDetents(detents) } else { content }
    }
}
