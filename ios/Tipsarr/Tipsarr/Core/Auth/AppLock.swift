import LocalAuthentication
import Observation
import SwiftUI

/// Optional Face ID lock (passcode as fallback). Locks at launch and after the app was in the
/// background for longer than the chosen delay. One lock protects every account on the iPhone.
@MainActor @Observable
final class AppLock {
    enum Delay: Int, CaseIterable, Sendable {
        case immediately = 0, minute1 = 60, minute5 = 300, minute15 = 900

        var title: LText {
            switch self {
            case .immediately: "m.faceid.immediately"
            case .minute1: "m.faceid.after_1"
            case .minute5: "m.faceid.after_5"
            case .minute15: "m.faceid.after_15"
            }
        }
    }

    private static let enabledKey = "faceIDLockEnabled"
    private static let delayKey = "faceIDLockDelay"

    private(set) var isEnabled: Bool
    var delay: Delay {
        didSet { UserDefaults.standard.set(delay.rawValue, forKey: Self.delayKey) }
    }

    private(set) var isLocked: Bool
    /// True while the system authentication sheet is on screen (the app is "inactive" then).
    private(set) var authenticating = false
    private(set) var failed = false
    private var leftAt: Date?

    init() {
        let enabled = UserDefaults.standard.bool(forKey: Self.enabledKey)
        isEnabled = enabled
        isLocked = enabled
        delay = Delay(rawValue: UserDefaults.standard.integer(forKey: Self.delayKey)) ?? .immediately
    }

    /// Icon of what unlocks the app on this iPhone.
    var symbol: String {
        let context = LAContext()
        _ = context.canEvaluatePolicy(.deviceOwnerAuthentication, error: nil)
        return context.biometryType == .touchID ? "touchid" : "faceid"
    }

    /// The device has a passcode (needed for any lock).
    var isAvailable: Bool {
        LAContext().canEvaluatePolicy(.deviceOwnerAuthentication, error: nil)
    }

    // MARK: Life cycle

    func scenePhaseChanged(_ phase: ScenePhase) {
        switch phase {
        case .background:
            if leftAt == nil, !isLocked { leftAt = .now }
        case .active:
            guard isEnabled, !isLocked, let leftAt else { return }
            self.leftAt = nil
            if delay == .immediately || Date.now.timeIntervalSince(leftAt) >= Double(delay.rawValue) { isLocked = true }
        default:
            break
        }
    }

    // MARK: Authentication

    func unlock() async {
        guard isLocked, !authenticating else { return }
        if await authenticate() { isLocked = false }
    }

    /// Turning the lock on needs a successful authentication first, so it cannot lock the person out.
    func enable() async -> Bool {
        guard await authenticate() else { return false }
        isEnabled = true
        UserDefaults.standard.set(true, forKey: Self.enabledKey)
        return true
    }

    func disable() {
        isEnabled = false
        isLocked = false
        leftAt = nil
        UserDefaults.standard.set(false, forKey: Self.enabledKey)
    }

    private func authenticate() async -> Bool {
        authenticating = true
        defer { authenticating = false }
        let context = LAContext()
        do {
            let ok = try await context.evaluatePolicy(.deviceOwnerAuthentication, localizedReason: L10n.string("m.lock.reason"))
            failed = !ok
            return ok
        } catch {
            failed = true
            return false
        }
    }
}
