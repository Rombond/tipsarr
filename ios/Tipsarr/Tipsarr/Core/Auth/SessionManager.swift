import Foundation
import Observation
import UIKit
import UserNotifications

/// What the root view shows. Launch flow of the behaviour spec, section 1.
enum SessionPhase: Equatable {
    case launching
    case connect
    case login(Server, prefillUsername: String?)
    case offline
    case failed(String)
    case updateRequired(minVersion: String)
    case sessionExpired(Account, Server)
    case ready(Account, Profile)
}

@MainActor @Observable
final class SessionManager {
    private(set) var phase: SessionPhase = .launching
    let accounts = AccountStore()
    /// Status of the active server, read at launch or sign-in.
    private(set) var serverStatus: ServerStatus?
    /// A link opened before the app was ready (or for another account); the tabs consume it.
    private(set) var pendingLink: DeepLink?

    /// Launch: resume the active account, or ask for a server.
    func start() async {
        guard let account = accounts.active else {
            phase = .connect
            return
        }
        await resume(account)
    }

    func retry() async {
        phase = .launching
        await start()
    }

    private func resume(_ account: Account) async {
        phase = .launching
        let server: Server
        do {
            let status = try await TipsarrAPI(serverURL: account.serverURL, token: nil).status()
            server = Server(url: account.serverURL, status: status)
        } catch {
            return fail(APIError.from(error))
        }
        if AppVersion.isOlder(TipsarrClient.appVersion, than: server.status.minAppVersion) {
            phase = .updateRequired(minVersion: server.status.minAppVersion)
            return
        }
        do {
            let profile = try await TipsarrAPI(serverURL: account.serverURL, token: account.token).me()
            serverStatus = server.status
            accounts.add(account)
            phase = .ready(account, profile)
        } catch let error as APIError where error.isUnauthorized {
            phase = .sessionExpired(account, server)
        } catch {
            fail(APIError.from(error))
        }
    }

    private func fail(_ error: APIError) {
        phase = error == .unreachable ? .offline : .failed(error.localizedMessage)
    }

    /// Connect screen: the address answers `/status` like a Tipsarr server.
    func connect(to url: URL) async throws -> Server {
        let status = try await TipsarrAPI(serverURL: url, token: nil).status()
        return Server(url: url, status: status)
    }

    func continueToLogin(_ server: Server, username: String? = nil) {
        if AppVersion.isOlder(TipsarrClient.appVersion, than: server.status.minAppVersion) {
            phase = .updateRequired(minVersion: server.status.minAppVersion)
        } else {
            phase = .login(server, prefillUsername: username)
        }
    }

    /// `rememberWithFaceID` keeps the login in the Keychain so Face ID can sign in again when the session ends.
    func signIn(server: Server, username: String, password: String, rememberWithFaceID: Bool = false) async throws {
        let api = TipsarrAPI(serverURL: server.url, token: nil)
        let result = try await api.signIn(username: username, password: password, deviceName: UIDevice.current.name)
        let account = Account(serverURL: server.url, userID: result.profile.id, name: result.profile.name,
                              token: result.token, lastUsed: .now)
        accounts.add(account)
        if rememberWithFaceID {
            CredentialStore.save(Credential(username: username, password: password), accountID: account.id)
        }
        serverStatus = server.status
        phase = .ready(account, result.profile)
    }

    /// Session ended: Face ID reads the stored login and signs in again. False when there is none or it failed.
    func reauthenticate(_ account: Account, server: Server) async -> Bool {
        guard let credential = await CredentialStore.load(accountID: account.id, reason: L10n.string("m.session.faceid_signin")) else { return false }
        do {
            try await signIn(server: server, username: credential.username, password: credential.password)
            return true
        } catch {
            // The password changed on the server: the stored one is useless now.
            if APIError.from(error).code == "invalid_credentials" { CredentialStore.delete(accountID: account.id) }
            return false
        }
    }

    /// Removes the account from the phone; the next one (or Connect) takes over.
    func signOut(_ account: Account) async {
        CredentialStore.delete(accountID: account.id)
        await PushManager.shared.detach(account: account)
        await TipsarrAPI(serverURL: account.serverURL, token: account.token).logout()
        accounts.remove(account)
        try? await UNUserNotificationCenter.current().setBadgeCount(0)
        await start()
    }

    func backToConnect() { phase = .connect }

    // MARK: Deep links

    /// Picks the account of the link's server (switching if needed), asks to sign in when there is none.
    func handle(_ url: URL) async {
        guard let link = DeepLink.parse(url) else { return }
        pendingLink = link
        let host = link.host
        if case .ready(let current, _) = phase, current.serverURL.host()?.lowercased() == host { return }
        if let account = accounts.accounts.first(where: { $0.serverURL.host()?.lowercased() == host }) {
            await switchTo(account)
        } else if let server = try? await connect(to: URL(string: "https://\(host)")!) {
            continueToLogin(server)
        }
    }

    func clearPendingLink() { pendingLink = nil }

    /// A tapped push alert: the event names the screen, the id the request or issue, the server is the active one.
    /// The alert names its server: the matching account is opened (switching if needed). An alert for a server
    /// that is not signed in on this phone is ignored. Without a server (older alerts) the active account is used.
    func openFromPush(_ tap: PushTap) async {
        let account: Account?
        if let server = tap.server?.lowercased(), !server.isEmpty {
            account = accounts.accounts.first { PushManager.serverKey($0.serverURL) == server }
                ?? accounts.accounts.first { $0.serverURL.host()?.lowercased() == server.split(separator: ":").first.map(String.init) }
        } else {
            account = accounts.active
        }
        guard let account, let host = account.serverURL.host()?.lowercased() else { return }
        let target: DeepLinkTarget = tap.event.hasPrefix("issue.") ? .issue(tap.id) : .request(tap.id)
        pendingLink = DeepLink(host: host, target: target)
        await switchTo(account)
    }

    /// Profile changed on the server (language, score source): keep the signed-in screens in sync.
    func update(profile: Profile) {
        if case .ready(let account, _) = phase { phase = .ready(account, profile) }
    }

    /// Switches to another stored account (no password asked).
    func switchTo(_ account: Account) async {
        guard account.id != accounts.activeID else { return }
        accounts.setActive(account.id)
        await resume(account)
    }

    /// Add account: same server first (login), the login screen offers another server.
    func beginAddAccount() {
        guard let account = accounts.active, let status = serverStatus else { return backToConnect() }
        phase = .login(Server(url: account.serverURL, status: status), prefillUsername: nil)
    }

    /// Leaves the add-account flow without signing in.
    func cancelAddAccount() async { await start() }
}
