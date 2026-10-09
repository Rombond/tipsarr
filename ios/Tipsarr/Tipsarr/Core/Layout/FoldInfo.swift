import SwiftUI

/// The vertical hinge of an unfolded iPhone Duo, in the coordinate space of the window.
/// Content must stay out of `minX - leadingMargin ... maxX + trailingMargin`.
struct Hinge: Equatable {
    var minX: CGFloat
    var maxX: CGFloat
    var leadingMargin: CGFloat
    var trailingMargin: CGFloat

    /// Width of the left pane (everything left of the hinge and its margin).
    var leftWidth: CGFloat { minX - leadingMargin }
    /// Where the right pane starts.
    var rightStart: CGFloat { maxX + trailingMargin }
}

enum FoldInfo {
    /// The active vertical fold region of the window, if there is one (iOS 27.1 and later).
    static func verticalHinge(_ proxy: GeometryProxy) -> Hinge? {
        guard #available(iOS 27.1, *) else { return nil }
        for region in proxy.reservedRegions(kind: .division) where region.isActive && region.frame.height > region.frame.width {
            return Hinge(minX: region.frame.minX, maxX: region.frame.maxX,
                         leadingMargin: region.margins.leading, trailingMargin: region.margins.trailing)
        }
        return nil
    }
}
