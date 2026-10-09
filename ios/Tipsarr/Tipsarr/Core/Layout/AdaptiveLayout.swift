import SwiftUI

/// How the five tabs are shown. Decided from the size of the window, never from the device:
/// iPhone, a narrow Split View or Stage Manager window use the system tab bar; a wide window
/// uses a sidebar when it is landscape and a floating bar at the bottom when it is portrait.
enum ShellStyle: Equatable {
    case system, bottomBar, sidebar
    /// Unfolded iPhone Duo: the list on the left of the hinge, the detail on the right.
    case dualPane(Hinge)

    static func style(size: CGSize, compact: Bool, hinge: Hinge?) -> ShellStyle {
        if let hinge { return .dualPane(hinge) }
        if compact { return .system }
        return size.width > size.height ? .sidebar : .bottomBar
    }
}

/// Layout kind of a screen's content area, from its own width (a screen can sit in a sidebar window,
/// a split view column or a narrow window).
enum ContentWidth {
    case compact, regular, wide

    init(_ width: CGFloat) {
        self = width < 620 ? .compact : (width < 900 ? .regular : .wide)
    }
}

extension Tokens.Palette {
    /// Sidebar background of the iPad shell (Penpot: #F7F7F8 light, #101010 dark). Not in tokens.json yet.
    static var sidebarColor: Color { Color.dynamic(light: 0xF7F7F8FF, dark: 0x101010FF) }
}

extension Tokens.Palette {
    var sidebar: Color { Self.sidebarColor }
}
