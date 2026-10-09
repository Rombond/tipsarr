import SwiftUI

/// Shown while the launch flow checks the server.
struct LaunchView: View {
    var body: some View {
        ZStack {
            Tokens.palette.bg.ignoresSafeArea()
            ProgressView().accessibilityLabel(Text("common.loading"))
        }
    }
}

struct SessionExpiredView: View {
    @Environment(SessionManager.self) private var session
    let account: Account
    let server: Server

    var body: some View {
        StateView(symbol: "lock.slash", title: "m.session.expired_title", message: "m.session.expired_body",
                  actionTitle: "m.session.sign_in") {
            session.continueToLogin(server, username: account.name)
        }
        .background(Tokens.palette.bg)
    }
}

struct UpdateRequiredView: View {
    let minVersion: String

    var body: some View {
        VStack(spacing: Tokens.Spacing.lg) {
            StateView(symbol: "arrow.down.app", title: "m.update.title", message: "m.update.body")
            if !minVersion.isEmpty {
                Text(verbatim: minVersion).font(.footnote).foregroundStyle(Tokens.palette.mutedFg)
            }
            if let url = AppStore.url {
                Link("m.update.open_store", destination: url).buttonStyle(.tipsarr(.primary))
            }
        }
        .padding(.bottom, Tokens.Spacing._3xl)
        .background(Tokens.palette.bg)
    }
}

enum AppStore {
    /// Set once the App Store listing exists (`https://apps.apple.com/app/id<ID>`); the button stays hidden until then.
    static let url: URL? = nil
}
