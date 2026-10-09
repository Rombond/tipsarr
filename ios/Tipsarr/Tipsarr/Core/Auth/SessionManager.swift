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

    func signIn(server: Server, username: String, password: String) async throws {
        let api = TipsarrAPI(serverURL: server.url, token: nil)
        let result = try await api.signIn(username: username, password: password, deviceName: UIDevice.current.name)
        let account = Account(serverURL: server.url, userID: result.profile.id, name: result.profile.name,
                              token: result.token, lastUsed: .now)
        accounts.add(account)
        serverStatus = server.status
        phase = .ready(account, result.profile)
    }

    /// Removes the account from the phone; the next one (or Connect) takes over.
    func signOut(_ account: Account) async {
        await TipsarrAPI(serverURL: account.serverURL, token: account.token).logout()
        accounts.remove(account)
        try? await UNUserNotificationCenter.current().setBadgeCount(0)
        await start()
    }

    func backToConnect() { phase = .connect }

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
