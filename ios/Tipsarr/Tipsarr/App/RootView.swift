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
        #if DEBUG
        // Debug aid for the iPhone Duo: print the fold (division) regions and the window size.
        .background {
            GeometryReader { proxy in
                Color.clear.task(id: proxy.size) {
                    if #available(iOS 27.1, *) {
                        let regions = proxy.reservedRegions(kind: .division, options: .includeInactive)
                        print("[fold] size=\(proxy.size) regions=\(regions.map { "\($0.frame) active=\($0.isActive) margins=\($0.margins)" })")
                    } else {
                        print("[fold] size=\(proxy.size) (no fold API)")
                    }
                }
            }
        }
        #endif
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
