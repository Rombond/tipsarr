import Foundation
import Security

/// One generic-password item per account (JSON of `Account`). No Keychain Sharing needed.
enum KeychainStore {
    private static let service = "com.brebond.tipsarr.account"

    static func all() -> [Account] {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecReturnData as String: true,
            kSecMatchLimit as String: kSecMatchLimitAll,
        ]
        var result: CFTypeRef?
        guard SecItemCopyMatching(query as CFDictionary, &result) == errSecSuccess,
              let items = result as? [Data] else { return [] }
        return items.compactMap { try? JSONDecoder().decode(Account.self, from: $0) }
    }

    @discardableResult
    static func save(_ account: Account) -> Bool {
        guard let data = try? JSONEncoder().encode(account) else { return false }
        let match: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: account.id,
        ]
        let update = SecItemUpdate(match as CFDictionary, [kSecValueData as String: data] as CFDictionary)
        if update == errSecSuccess { return true }
        var add = match
        add[kSecValueData as String] = data
        add[kSecAttrAccessible as String] = kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly
        return SecItemAdd(add as CFDictionary, nil) == errSecSuccess
    }

    static func delete(_ account: Account) {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: account.id,
        ]
        SecItemDelete(query as CFDictionary)
    }
}
