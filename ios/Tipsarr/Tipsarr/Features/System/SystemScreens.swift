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
    @State private var hasFaceID: Bool
    @State private var trying = false

    init(account: Account, server: Server) {
        self.account = account
        self.server = server
        _hasFaceID = State(initialValue: CredentialStore.exists(accountID: account.id))
    }

    var body: some View {
        VStack(spacing: Tokens.Spacing.lg) {
            StateView(symbol: hasFaceID ? "faceid" : "lock.slash", title: "m.session.expired_title", message: "m.session.expired_body")
            VStack(spacing: Tokens.Spacing.sm) {
                if hasFaceID {
                    Button { Task { await faceID() } } label: {
                        if trying {
                            HStack(spacing: Tokens.Spacing.sm) { ProgressView(); Text("m.session.signing_in_faceid") }
                        } else {
                            Label { Text("m.session.faceid_signin") } icon: { Image(systemName: "faceid") }
                        }
                    }
                    .buttonStyle(.tipsarr(.primary, fullWidth: true))
                    .disabled(trying)
                }
                Button { session.continueToLogin(server, username: account.name) } label: {
                    Text(hasFaceID ? "m.session.use_password" : "m.session.sign_in")
                }
                .buttonStyle(.tipsarr(hasFaceID ? .ghost : .primary, fullWidth: true))
            }
            .padding(.horizontal, Tokens.Spacing._2xl)
            .padding(.bottom, Tokens.Spacing._3xl)
        }
        .background(Tokens.palette.bg)
        // The session ended while the person is away: ask for Face ID right away.
        .task { if hasFaceID { await faceID() } }
    }

    private func faceID() async {
        guard !trying else { return }
        trying = true
        defer { trying = false }
        if await session.reauthenticate(account, server: server) { return }
        hasFaceID = CredentialStore.exists(accountID: account.id)
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
