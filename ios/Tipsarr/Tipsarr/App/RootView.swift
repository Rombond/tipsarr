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
                StateView(symbol: "exclamationmark.triangle", title: LText.verbatim(message),
                          actionTitle: "common.retry") { Task { await session.retry() } }
            case .updateRequired(let minVersion):
                UpdateRequiredView(minVersion: minVersion)
            case .sessionExpired(let account, let server):
                SessionExpiredView(account: account, server: server)
            case .ready(let account, let profile):
                MainTabView(account: account, profile: profile)
            }
        }
        .environment(session)
        .animation(.easeOut(duration: Tokens.Motion.normal), value: session.phase)
        .task { await session.start() }
    }
}
