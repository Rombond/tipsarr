import SwiftUI

/// Progress of a download pushed by the server.
struct LiveProgress: Equatable, Sendable {
    var percent: Int
    var etaSeconds: Int
}

/// What changed on the server, as counters screens can watch with `.onChange`.
/// The stream runs only while the app is in the foreground and a person is signed in.
@MainActor @Observable
final class LiveUpdates {
    /// A request changed (created, approved, declined, deleted, status change) or a title became available.
    private(set) var requestsTick = 0
    private(set) var suggestionsTick = 0
    private(set) var issuesTick = 0
    /// Latest progress per request id; cleared when the request changes state.
    private(set) var progress: [String: LiveProgress] = [:]
    private(set) var connected = false

    @ObservationIgnored private var hasConnectedOnce = false
    @ObservationIgnored private var task: Task<Void, Never>?
    @ObservationIgnored private var runningFor: String?

    func start(server: URL, token: String) {
        let key = server.absoluteString + token
        guard runningFor != key else { return }
        stop()
        runningFor = key
        task = Task { [weak self] in
            await EventStream.run(
                server: server, token: token,
                onConnected: { self?.didConnect() },
                onEvent: { self?.handle($0) }
            )
        }
    }

    func stop() {
        task?.cancel()
        task = nil
        runningFor = nil
        connected = false
    }

    /// After every reconnection everything is refetched: events sent while away are not replayed to this client.
    /// The very first connection does not, the screens just loaded their data themselves.
    private func didConnect() {
        connected = true
        #if DEBUG
        print("[live] connected (reconnect: \(hasConnectedOnce))")
        #endif
        defer { hasConnectedOnce = true }
        guard hasConnectedOnce else { return }
        requestsTick += 1
        suggestionsTick += 1
        issuesTick += 1
    }

    private func handle(_ event: LiveEvent) {
        #if DEBUG
        print("[live] \(event.type) \(event.itemID ?? "") \(event.percent.map { "\($0)%" } ?? "")")
        #endif
        switch event.type {
        case "request.progress":
            if let id = event.itemID, let percent = event.percent {
                progress[id] = LiveProgress(percent: percent, etaSeconds: event.etaSeconds ?? 0)
            }
        case "request.updated":
            if let id = event.itemID { progress[id] = nil }
            requestsTick += 1
        case "media.available":
            requestsTick += 1
        case "suggestions.updated":
            suggestionsTick += 1
        case "issue.updated":
            issuesTick += 1
        default:
            break
        }
    }
}

private struct LiveUpdatesKey: EnvironmentKey {
    static let defaultValue: LiveUpdates? = nil
}

extension EnvironmentValues {
    var liveUpdates: LiveUpdates? {
        get { self[LiveUpdatesKey.self] }
        set { self[LiveUpdatesKey.self] = newValue }
    }
}
