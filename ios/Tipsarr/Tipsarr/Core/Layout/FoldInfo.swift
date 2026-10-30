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

/// The horizontal hinge of an iPhone Duo half folded like a laptop: content above `top`, controls below `bottom`
/// (window coordinates, margins included).
struct LaptopFold: Equatable {
    var top: CGFloat
    var bottom: CGFloat
}

enum FoldInfo {
    /// The vertical fold region of the window when the Duo is unfolded (iOS 27.1 and later). Measured in the
    /// simulator: the region exists with a 40 pt frame and 20 pt margins on both sides while unfolded, but its
    /// `isActive` flag is false when the device lies flat, so inactive regions count too.
    static func verticalHinge(_ proxy: GeometryProxy) -> Hinge? {
        guard #available(iOS 27.1, *) else { return nil }
        for region in proxy.reservedRegions(kind: .division, options: .includeInactive) where region.frame.height > region.frame.width && region.frame.width > 0 {
            return Hinge(minX: region.frame.minX, maxX: region.frame.maxX,
                         leadingMargin: region.margins.leading, trailingMargin: region.margins.trailing)
        }
        return nil
    }

    /// The fold of a Duo held sideways and half folded (laptop posture): a horizontal division region that is active.
    /// Lying flat the region is inactive, and held upright the hinge is vertical, so both give nil.
    static func laptopFold(_ proxy: GeometryProxy) -> LaptopFold? {
        guard #available(iOS 27.1, *) else { return nil }
        for region in proxy.reservedRegions(kind: .division) where region.isActive && region.frame.width > region.frame.height && region.frame.height > 0 {
            return LaptopFold(top: region.frame.minY - region.margins.top, bottom: region.frame.maxY + region.margins.bottom)
        }
        return nil
    }
}
