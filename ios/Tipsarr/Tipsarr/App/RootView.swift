import SwiftUI

struct RootView: View {
    @State private var session = SessionManager()
    @Environment(AppLock.self) private var lock
    @Environment(\.scenePhase) private var scenePhase

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
        .overlay {
            if lock.isLocked {
                LockView(lock: lock).transition(.opacity)
            } else if lock.isEnabled && scenePhase != .active && !lock.authenticating {
                PrivacyCover()
            }
        }
        .animation(.easeOut(duration: Tokens.Motion.fast), value: lock.isLocked)
        .onChange(of: scenePhase) { _, phase in lock.scenePhaseChanged(phase) }
        .animation(.easeOut(duration: Tokens.Motion.normal), value: session.phase)
        .task { await session.start() }
    }
}
