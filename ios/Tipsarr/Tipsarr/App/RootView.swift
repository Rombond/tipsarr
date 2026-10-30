import SwiftUI

struct RootView: View {
    @State private var session = SessionManager()
    @State private var push = PushManager.shared

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
            case .failed(let message, let detail):
                FailedView(message: message, detail: detail)
            case .updateRequired(let minVersion):
                UpdateRequiredView(minVersion: minVersion)
            case .sessionExpired(let account, let server):
                SessionExpiredView(account: account, server: server)
            case .ready(let account, let profile):
                MainTabView(account: account, profile: profile)
            }
        }
        .environment(session)
        .onOpenURL { url in Task { await session.handle(url) } }
        .animation(.easeOut(duration: Tokens.Motion.normal), value: session.phase)
        .task { await session.start() }
        // A tapped alert opens its request or issue (also when the tap launched the app).
        .task(id: push.tap) {
            guard let tap = push.tap else { return }
            await session.openFromPush(tap)
            push.tap = nil
        }
    }
}
