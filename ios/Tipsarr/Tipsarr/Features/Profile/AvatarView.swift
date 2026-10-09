import SwiftUI

/// Round picture of a user, initials when there is none.
struct AvatarView: View {
    let userID: String
    let name: String
    var size: CGFloat = 72
    /// Changes when the picture changes, so the cached copy is not reused.
    var version = 0

    var body: some View {
        RemoteImage(serverPath: "/api/v1/users/\(userID)/avatar?v=\(version)") {
            Text(verbatim: String(name.prefix(1)).uppercased())
                .font(.system(size: size * 0.4, weight: .semibold))
                .foregroundStyle(Tokens.palette.mutedFg)
        }
        .frame(width: size, height: size)
        .background(Tokens.palette.muted)
        .clipShape(.circle)
        .accessibilityHidden(true)
    }
}
