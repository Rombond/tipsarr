import Foundation

/// Permission flags, the same values as `backend/internal/perms` (Seerr's numbers). A person's `effectivePermissions`
/// is the sum of the flags they have; admins have every flag.
struct Permission: OptionSet, Sendable, Hashable {
    let rawValue: Int64

    static let manageSettings = Permission(rawValue: 4)
    static let manageUsers = Permission(rawValue: 8)
    static let manageRequests = Permission(rawValue: 16)
    static let request = Permission(rawValue: 32)
    static let autoApprove = Permission(rawValue: 128)
    static let autoApproveMovie = Permission(rawValue: 256)
    static let autoApproveTV = Permission(rawValue: 512)
    static let requestAdvanced = Permission(rawValue: 8192)
    static let requestView = Permission(rawValue: 16384)
    static let requestMovie = Permission(rawValue: 262144)
    static let requestTV = Permission(rawValue: 524288)
    static let manageIssues = Permission(rawValue: 1048576)
    static let viewIssues = Permission(rawValue: 2097152)
    static let createIssues = Permission(rawValue: 4194304)

    func contains(_ flags: [Permission]) -> Bool { flags.allSatisfy { contains($0) } }
}

/// A person's own request limits; nil = the default applies, 0 = no limit.
struct UserQuota: Sendable, Equatable, Codable {
    var movieLimit: Int?
    var movieDays: Int?
    var tvLimit: Int?
    var tvDays: Int?
}

/// One flag of the catalog from `GET /admin/permissions`.
struct PermissionFlag: Sendable, Identifiable, Hashable {
    var name: String
    var value: Int64
    var group: String
    var implies: [String]
    var reserved: Bool
    var id: String { name }
}

/// What new people start with.
struct PermissionDefaults: Sendable, Equatable {
    var permissions: Int64
    var movieQuotaLimit: Int
    var movieQuotaDays: Int
    var tvQuotaLimit: Int
    var tvQuotaDays: Int
}

struct PermissionCatalog: Sendable {
    var flags: [PermissionFlag]
    var defaults: PermissionDefaults

    /// Switches on the flags that come with another one, as the server does when it saves a mask.
    func normalize(_ mask: Int64) -> Int64 {
        var m = mask
        let byName = Dictionary(uniqueKeysWithValues: flags.map { ($0.name, $0.value) })
        for f in flags where m & f.value != 0 {
            for name in f.implies { m |= byName[name] ?? 0 }
        }
        return m
    }

    /// The flags that are on only because another flag implies them (they cannot be switched off alone).
    func forced(_ mask: Int64) -> Int64 {
        var out: Int64 = 0
        let byName = Dictionary(uniqueKeysWithValues: flags.map { ($0.name, $0.value) })
        for f in flags where mask & f.value != 0 {
            for name in f.implies { out |= byName[name] ?? 0 }
        }
        return out
    }
}

struct QuotaWindow: Sendable, Equatable {
    var days: Int
    var limit: Int
    var used: Int
    var remaining: Int?
    var restricted: Bool
    var resetsAt: Date?
}

struct Quota: Sendable, Equatable {
    var movie: QuotaWindow
    var tv: QuotaWindow

    func window(_ type: MediaType) -> QuotaWindow { type == .movie ? movie : tv }
}
