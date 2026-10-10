import Foundation
import Observation
import UIKit
import UserNotifications

/// What a tapped notification asks for: the event name and the id of the request or issue.
struct PushTap: Equatable, Sendable {
    var event: String
    var id: String
}

/// Push registration of this install: asks the system for permission and a token, tells the active
/// server about it (`PUT /me/devices/current`) and keeps the notification toggles. The server only
/// receives a token; alerts carry an event and an id, the app fetches the details when opened.
@MainActor @Observable
final class PushManager {
    static let shared = PushManager()

    /// Notification categories, a bitmask shared with the server.
    enum Category: Int, CaseIterable {
        case requests = 1, admin = 2, issues = 4
    }

    private(set) var authorization: UNAuthorizationStatus = .notDetermined
    /// The server knows this device; `categories` is its copy of the toggles.
    private(set) var registered = false
    private(set) var categories = Category.requests.rawValue | Category.admin.rawValue | Category.issues.rawValue
    /// Set by the notification delegate when the user taps an alert; the root view consumes it.
    var tap: PushTap?

    private var token: String?
    private var account: Account?
    private var serverAllowsPush = false
    private var api: TipsarrAPI? { account.map { TipsarrAPI(serverURL: $0.serverURL, token: $0.token) } }

    private static let askedKey = "push.askedForPermission"
    private func optOutKey(_ account: Account) -> String { "push.optOut.\(account.id)" }

    var optedOut: Bool {
        guard let account else { return false }
        return UserDefaults.standard.bool(forKey: optOutKey(account))
    }

    /// True when the server has push, and this device is registered and allowed to show alerts.
    var isOn: Bool { registered && isAuthorized }
    var isAuthorized: Bool { authorization == .authorized || authorization == .provisional || authorization == .ephemeral }
    var isDenied: Bool { authorization == .denied }

    // MARK: Lifecycle

    /// Called when an account is signed in and ready. Registers silently when the user already allowed alerts.
    func attach(account: Account, pushAvailable: Bool) async {
        self.account = account
        serverAllowsPush = pushAvailable
        registered = false
        await refreshAuthorization()
        guard pushAvailable, isAuthorized, !optedOut else { return }
        UIApplication.shared.registerForRemoteNotifications()
        await syncRegistration()
    }

    /// Before sign-out: stop alerts for this account on this device.
    func detach(account: Account) async {
        _ = try? await TipsarrAPI(serverURL: account.serverURL, token: account.token).unregisterDevice()
        if self.account?.id == account.id { registered = false }
    }

    func didRegister(deviceToken data: Data) {
        token = data.map { String(format: "%02x", $0) }.joined()
        Task { await syncRegistration() }
    }

    func didFailToRegister(_ error: Error) {
        #if DEBUG
        print("[push] registration failed: \(error)")
        #endif
    }

    // MARK: Asking

    /// First request created: the right moment to ask, once. Does nothing when the server has no push.
    func askAfterFirstRequest() async {
        guard serverAllowsPush, !UserDefaults.standard.bool(forKey: Self.askedKey) else { return }
        await refreshAuthorization()
        guard authorization == .notDetermined else { return }
        UserDefaults.standard.set(true, forKey: Self.askedKey)
        await turnOn()
    }

    /// Settings switch: asks for permission when needed and registers.
    func turnOn() async {
        guard let account else { return }
        UserDefaults.standard.set(false, forKey: optOutKey(account))
        await refreshAuthorization()
        if authorization == .notDetermined {
            _ = try? await UNUserNotificationCenter.current().requestAuthorization(options: [.alert, .sound, .badge])
            UserDefaults.standard.set(true, forKey: Self.askedKey)
            await refreshAuthorization()
        }
        guard isAuthorized else { return }
        UIApplication.shared.registerForRemoteNotifications()
        await syncRegistration()
    }

    func turnOff() async {
        guard let account else { return }
        UserDefaults.standard.set(true, forKey: optOutKey(account))
        _ = try? await api?.unregisterDevice()
        registered = false
    }

    func openSystemSettings() {
        if let url = URL(string: UIApplication.openNotificationSettingsURLString) { UIApplication.shared.open(url) }
    }

    func set(_ category: Category, on: Bool) async {
        let value = on ? categories | category.rawValue : categories & ~category.rawValue
        guard value != categories else { return }
        let before = categories
        categories = value
        do {
            try await register(categories: value)
        } catch {
            categories = before
        }
    }

    // MARK: Server

    private func refreshAuthorization() async {
        authorization = await UNUserNotificationCenter.current().notificationSettings().authorizationStatus
    }

    /// Sends the token (and language) to the server; keeps the toggles the server already has.
    private func syncRegistration() async {
        guard serverAllowsPush, isAuthorized, !optedOut, token != nil else { return }
        try? await register(categories: nil)
    }

    private func register(categories wanted: Int?) async throws {
        guard let api, let token else { return }
        let language = AppLanguage.locale.language.languageCode?.identifier ?? "en"
        let result = try await api.registerDevice(token: token, sandbox: Self.usesSandbox, language: language, categories: wanted)
        categories = result.categories
        registered = true
    }

    /// Xcode runs are signed with a development profile and get sandbox tokens; TestFlight and App Store
    /// builds have no embedded profile and get production tokens. The relay needs to know which.
    static var usesSandbox: Bool {
        guard let url = Bundle.main.url(forResource: "embedded", withExtension: "mobileprovision"),
              let data = try? Data(contentsOf: url) else { return false }
        let text = String(decoding: data, as: UTF8.self)
        guard let range = text.range(of: "<key>aps-environment</key>") else { return false }
        return text[range.upperBound...].prefix(60).contains("development")
    }
}
