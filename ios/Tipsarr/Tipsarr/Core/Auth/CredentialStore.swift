import Foundation
import LocalAuthentication
import Security

/// The Jellyfin login of an account, kept so Face ID can sign the person back in when the session ends.
/// The item is protected by the Keychain itself (Face ID or passcode to read it, this device only,
/// never in backups): the password is never readable without the person.
struct Credential: Codable, Sendable {
    var username: String
    var password: String
}

enum CredentialStore {
    private static let service = "com.brebond.tipsarr.credential"

    private static func base(_ accountID: String) -> [String: Any] {
        [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: accountID,
        ]
    }

    /// Needs a device passcode; returns false when the Keychain refuses.
    @discardableResult
    static func save(_ credential: Credential, accountID: String) -> Bool {
        guard let data = try? JSONEncoder().encode(credential),
              let access = SecAccessControlCreateWithFlags(nil, kSecAttrAccessibleWhenPasscodeSetThisDeviceOnly, .userPresence, nil) else { return false }
        delete(accountID: accountID)
        var item = base(accountID)
        item[kSecValueData as String] = data
        item[kSecAttrAccessControl as String] = access
        return SecItemAdd(item as CFDictionary, nil) == errSecSuccess
    }

    /// Is a login stored? Does not ask for Face ID.
    static func exists(accountID: String) -> Bool {
        let context = LAContext()
        context.interactionNotAllowed = true
        var query = base(accountID)
        query[kSecUseAuthenticationContext as String] = context
        query[kSecMatchLimit as String] = kSecMatchLimitOne
        let status = SecItemCopyMatching(query as CFDictionary, nil)
        return status == errSecSuccess || status == errSecInteractionNotAllowed
    }

    /// Shows the Face ID prompt; nil when cancelled or failed.
    static func load(accountID: String, reason: String) async -> Credential? {
        await Task.detached {
            let context = LAContext()
            context.localizedReason = reason
            var query = base(accountID)
            query[kSecReturnData as String] = true
            query[kSecMatchLimit as String] = kSecMatchLimitOne
            query[kSecUseAuthenticationContext as String] = context
            var result: CFTypeRef?
            guard SecItemCopyMatching(query as CFDictionary, &result) == errSecSuccess, let data = result as? Data else { return nil }
            return try? JSONDecoder().decode(Credential.self, from: data)
        }.value
    }

    static func delete(accountID: String) {
        SecItemDelete(base(accountID) as CFDictionary)
    }
}
