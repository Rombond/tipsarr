import ActivityKit
import Foundation
import Observation
import UIKit

/// Follows requests as Live Activities. The app starts the activity, sends its push token to the server
/// (`PUT /requests/{id}/follow`), and the server pushes the state changes through the relay to APNs.
/// Ending an activity (the user swipes it away, or Stop following) removes the follow on the server.
@MainActor @Observable
final class LiveFollowManager {
    static let shared = LiveFollowManager()

    /// Request ids followed by the current account on this device.
    private(set) var followed: Set<String> = []
    /// The server supports following.
    private(set) var available = false

    private var account: Account?
    private var observers: [String: Task<Void, Never>] = [:]
    private var api: TipsarrAPI? { account.map { TipsarrAPI(serverURL: $0.serverURL, token: $0.token) } }

    private static let enabledKey = "liveFollow.enabled"

    /// Settings switch, on by default.
    var enabled: Bool {
        get {
            access(keyPath: \.enabled)
            return UserDefaults.standard.object(forKey: Self.enabledKey) as? Bool ?? true
        }
        set {
            withMutation(keyPath: \.enabled) { UserDefaults.standard.set(newValue, forKey: Self.enabledKey) }
            if !newValue { Task { await unfollowAll() } }
        }
    }

    /// Follow controls are shown.
    var isOn: Bool { available && enabled }

    func isFollowing(_ id: String) -> Bool { followed.contains(id) }

    /// Final requests cannot be followed.
    func canFollow(_ record: RequestRecord) -> Bool { isOn && record.isOpen }

    // MARK: Lifecycle

    /// Called with PushManager.attach: picks up the activities still running for this account and drops
    /// the ones the server no longer follows (or follows the server forgot).
    func attach(account: Account, liveFollowAvailable: Bool) async {
        if self.account?.id != account.id { stopObserving() }
        self.account = account
        available = liveFollowAvailable
        let mine = activities(of: account)
        guard liveFollowAvailable, enabled else {
            for activity in mine { await end(activity, unfollow: liveFollowAvailable) }
            return
        }
        let server = (try? await api?.listFollows()).map { Dictionary($0.map { ($0.requestId, $0) }, uniquingKeysWith: { a, _ in a }) }
        for activity in mine {
            let id = activity.requestId
            if let server, server[id] == nil {
                // The server ended the follow (expired, final, deleted) while the push may have been lost.
                await Self.endActivity(activity.id)
                removePoster(id)
                continue
            }
            if let state = server?[id] { await Self.updateActivity(activity.id, state: state.contentState) }
            observe(activity)
        }
    }

    /// Before sign-out: ends this account's activities and their follows.
    func detach(account: Account) async {
        let api = TipsarrAPI(serverURL: account.serverURL, token: account.token)
        for activity in activities(of: account) {
            observers[activity.requestId]?.cancel()
            await Self.endActivity(activity.id)
            _ = try? await api.unfollowRequest(id: activity.requestId)
            removePoster(activity.requestId)
        }
        if self.account?.id == account.id {
            stopObserving()
            self.account = nil
        }
    }

    // MARK: Follow

    /// Starts the Live Activity and registers it on the server. Returns the text to show in a toast.
    func follow(_ record: RequestRecord) async -> (text: String, failed: Bool) {
        guard let account, isOn else { return ("", true) }
        guard ActivityAuthorizationInfo().areActivitiesEnabled else {
            return (L10n.string("m.requests.follow_live_off"), true)
        }
        if followed.contains(record.id) { return (L10n.string("m.requests.follow_started", record.title), false) }
        await writePoster(record, account: account)
        let attributes = RequestActivityAttributes(
            requestId: record.id,
            accountId: account.id,
            server: PushManager.serverKey(account.serverURL),
            title: record.title,
            mediaType: record.type.rawValue,
            labels: Self.labels
        )
        let state = Self.contentState(record)
        do {
            let activity = try Activity.request(attributes: attributes, content: ActivityContent(state: state, staleDate: nil), pushType: .token)
            let ref = Ref(id: activity.id, requestId: record.id, accountId: account.id, server: attributes.server)
            // Say "following" only once the server took the token: otherwise the activity would never move.
            guard let token = await Self.firstToken(activity.id) else {
                await end(ref, unfollow: false)
                return (L10n.string("m.requests.follow_live_off"), true)
            }
            if let failure = await register(ref, token: token) {
                await end(ref, unfollow: false)
                return (failure, true)
            }
            observe(ref)
            return (L10n.string("m.requests.follow_started", record.title), false)
        } catch {
            removePoster(record.id)
            return (L10n.string("m.requests.follow_live_off"), true)
        }
    }

    /// Ends the activity at once and removes the follow on the server.
    func unfollow(_ id: String) async {
        guard let account else { return }
        if let activity = activities(of: account).first(where: { $0.requestId == id }) {
            await end(activity, unfollow: true)
        } else {
            followed.remove(id)
            _ = try? await api?.unfollowRequest(id: id)
        }
    }

    private func unfollowAll() async {
        guard let account else { return }
        for activity in activities(of: account) { await end(activity, unfollow: true) }
    }

    // MARK: Activities

    /// What the manager keeps of an activity; `Activity` itself is not Sendable, so it is looked up by id
    /// inside the nonisolated helpers below.
    struct Ref: Sendable {
        var id: String
        var requestId: String
        var accountId: String
        var server: String
    }

    private func activities(of account: Account) -> [Ref] {
        Activity<RequestActivityAttributes>.activities
            .filter { $0.attributes.accountId == account.id }
            .map { Ref(id: $0.id, requestId: $0.attributes.requestId, accountId: $0.attributes.accountId, server: $0.attributes.server) }
    }

    private nonisolated static func find(_ id: String) -> Activity<RequestActivityAttributes>? {
        Activity<RequestActivityAttributes>.activities.first { $0.id == id }
    }

    private nonisolated static func endActivity(_ id: String) async {
        await find(id)?.end(nil, dismissalPolicy: .immediate)
    }

    /// Applies a server state when it is newer than the one shown.
    private nonisolated static func updateActivity(_ id: String, state: RequestActivityAttributes.ContentState) async {
        guard let activity = find(id), state.updatedAt > activity.content.state.updatedAt else { return }
        await activity.update(ActivityContent(state: state, staleDate: nil))
    }

    private nonisolated static func pushTokens(_ id: String) -> AsyncStream<String> {
        AsyncStream { continuation in
            let task = Task {
                if let activity = find(id) {
                    for await data in activity.pushTokenUpdates {
                        continuation.yield(data.map { String(format: "%02x", $0) }.joined())
                    }
                }
                continuation.finish()
            }
            continuation.onTermination = { _ in task.cancel() }
        }
    }

    /// Returns once the activity is ended or dismissed (or gone).
    private nonisolated static func waitForEnd(_ id: String) async {
        guard let activity = find(id) else { return }
        for await state in activity.activityStateUpdates where state == .dismissed || state == .ended { return }
    }

    private func end(_ activity: Ref, unfollow: Bool) async {
        let id = activity.requestId
        observers[id]?.cancel()
        observers[id] = nil
        followed.remove(id)
        await Self.endActivity(activity.id)
        if unfollow { _ = try? await api?.unfollowRequest(id: id) }
        removePoster(id)
    }

    private func stopObserving() {
        observers.values.forEach { $0.cancel() }
        observers = [:]
        followed = []
    }

    /// Sends each push token to the server and ends the follow when the activity goes away.
    private func observe(_ activity: Ref) {
        let id = activity.requestId
        observers[id]?.cancel()
        followed.insert(id)
        let tokens = Task { [weak self] in
            for await token in Self.pushTokens(activity.id) {
                await self?.refresh(activity, token: token)
            }
        }
        observers[id] = Task { [weak self] in
            await Self.waitForEnd(activity.id)
            tokens.cancel()
            guard !Task.isCancelled else { return }
            await self?.activityGone(activity)
        }
    }

    /// Sends a token to the server. Returns the error text when the follow cannot work (404, 409, 422) or the
    /// server could not be reached; nil when it is registered.
    @discardableResult
    private func register(_ activity: Ref, token: String) async -> String? {
        guard let api, activity.accountId == account?.id else { return nil }
        do {
            let state = try await api.followRequest(id: activity.requestId, pushToken: token, activityId: activity.id,
                                                    sandbox: PushManager.usesSandbox, server: activity.server)
            await Self.updateActivity(activity.id, state: state.contentState)
            return nil
        } catch {
            return APIError.from(error).localizedMessage
        }
    }

    /// A token refresh in the background: on a refusal the activity would never update, so it is ended.
    private func refresh(_ activity: Ref, token: String) async {
        guard let api, activity.accountId == account?.id else { return }
        do {
            let state = try await api.followRequest(id: activity.requestId, pushToken: token, activityId: activity.id,
                                                    sandbox: PushManager.usesSandbox, server: activity.server)
            await Self.updateActivity(activity.id, state: state.contentState)
        } catch {
            let apiError = APIError.from(error)
            // 404, 409 (final, no push, too many) and 422 (bad token or server): the activity would never update.
            if apiError.status == 409 || apiError.status == 422 || apiError.status == 404 {
                await end(activity, unfollow: false)
                ToastBridge.show(apiError.localizedMessage)
            }
        }
    }

    /// The activity's first push token, or nil when none comes within the timeout.
    private nonisolated static func firstToken(_ id: String, timeout: Duration = .seconds(15)) async -> String? {
        await withTaskGroup(of: String?.self) { group in
            group.addTask {
                for await token in pushTokens(id) { return token }
                return nil
            }
            group.addTask {
                try? await Task.sleep(for: timeout)
                return nil
            }
            let first = await group.next() ?? nil
            group.cancelAll()
            return first
        }
    }

    private func activityGone(_ activity: Ref) async {
        let id = activity.requestId
        observers[id] = nil
        followed.remove(id)
        if activity.accountId == account?.id { _ = try? await api?.unfollowRequest(id: id) }
        removePoster(id)
    }

    // MARK: State

    /// Stage labels in the app language; the extension shows them as they are.
    private static var labels: [String: String] {
        let stages = ["requested", "approved", "searching", "downloading", "available", "declined", "failed"]
        return Dictionary(uniqueKeysWithValues: stages.map { ($0, L10n.string("state.\($0)")) })
    }

    static func contentState(_ record: RequestRecord) -> RequestActivityAttributes.ContentState {
        let stage: String
        switch record.state {
        case .partial: stage = "searching"
        default: stage = record.state.rawValue
        }
        let percent = record.state == .available ? 100 : (record.state == .downloading ? record.progressPercent ?? 0 : 0)
        return .init(stage: stage, percent: max(0, min(100, percent)), etaSeconds: max(0, record.etaSeconds ?? 0),
                     updatedAt: Int(Date.now.timeIntervalSince1970))
    }

    // MARK: Poster (best effort, App Group)

    private func writePoster(_ record: RequestRecord, account: Account) async {
        guard let path = record.posterPath, let file = LivePoster.url(requestId: record.id),
              let url = ImageSource(serverURL: account.serverURL, token: account.token).tmdbURL(size: .w185, path: path),
              let image = try? await ImageLoader.shared.image(url: url, token: account.token),
              let data = image.jpegData(compressionQuality: 0.8) else { return }
        try? FileManager.default.createDirectory(at: file.deletingLastPathComponent(), withIntermediateDirectories: true)
        try? data.write(to: file, options: .atomic)
    }

    private func removePoster(_ id: String) {
        if let file = LivePoster.url(requestId: id) { try? FileManager.default.removeItem(at: file) }
    }
}

/// Lets the manager show an error toast outside a view (a token refresh in the background).
@MainActor
enum ToastBridge {
    static weak var center: ToastCenter?
    static func show(_ text: String) { center?.show(text, kind: .error) }
}
