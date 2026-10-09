import Foundation
import Observation

/// Accounts in the Keychain, the active one in `UserDefaults`.
@MainActor @Observable
final class AccountStore {
    private static let activeKey = "activeAccountID"

    private(set) var accounts: [Account]
    private(set) var activeID: String?

    var active: Account? { accounts.first { $0.id == activeID } }

    init() {
        let stored = KeychainStore.all().sorted { $0.lastUsed > $1.lastUsed }
        let saved = UserDefaults.standard.string(forKey: Self.activeKey)
        accounts = stored
        activeID = stored.contains { $0.id == saved } ? saved : stored.first?.id
    }

    /// Adds or replaces an account and makes it the active one.
    func add(_ account: Account) {
        var account = account
        account.lastUsed = .now
        KeychainStore.save(account)
        accounts.removeAll { $0.id == account.id }
        accounts.insert(account, at: 0)
        setActive(account.id)
    }

    func setActive(_ id: String?) {
        activeID = id
        UserDefaults.standard.set(id, forKey: Self.activeKey)
    }

    func remove(_ account: Account) {
        KeychainStore.delete(account)
        accounts.removeAll { $0.id == account.id }
        if activeID == account.id { setActive(accounts.first?.id) }
    }
}
