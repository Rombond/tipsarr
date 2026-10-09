import SwiftUI

struct RootView: View {
    @State private var session = SessionManager()

    var body: some View {
        Group {
            switch session.phase {
            case .launching:
                LaunchView()
            case .connect:
                ConnectView()
            case .login(let server, let username):
                LoginView(server: server, prefillUsername: username)
            case .offline:
                StateView.offline { Task { await session.retry() } }
            case .failed(let message):
                StateView(symbol: "exclamationmark.triangle", title: LocalizedStringResource(stringLiteral: message),
                          actionTitle: "common.retry") { Task { await session.retry() } }
            case .updateRequired(let minVersion):
                UpdateRequiredView(minVersion: minVersion)
            case .sessionExpired(let account, let server):
                SessionExpiredView(account: account, server: server)
            case .ready(let account, let profile):
                SignedInPlaceholder(account: account, profile: profile)
            }
        }
        .environment(session)
        .animation(.easeOut(duration: Tokens.Motion.normal), value: session.phase)
        .task { await session.start() }
    }
}

/// Step 3 replaces this with the tab bar.
private struct SignedInPlaceholder: View {
    @Environment(SessionManager.self) private var session
    let account: Account
    let profile: Profile

    var body: some View {
        VStack(spacing: Tokens.Spacing.lg) {
            Text(verbatim: profile.name).font(.title.weight(.bold))
            Text(verbatim: account.serverURL.absoluteString).foregroundStyle(Tokens.palette.mutedFg)
            Button("m.settings.sign_out_confirm") { Task { await session.signOut(account) } }
                .buttonStyle(.tipsarr(.secondary))
        }
        .padding()
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .background(Tokens.palette.bg)
    }
}
