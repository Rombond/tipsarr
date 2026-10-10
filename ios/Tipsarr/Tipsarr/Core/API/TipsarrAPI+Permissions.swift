import Foundation

extension TipsarrAPI {
    func myQuota() async throws -> Quota {
        try await load { client in
            guard case .ok(let ok) = try await client.myQuota() else { throw APIError.unexpected }
            let body = try ok.body.json
            func window(_ w: Components.Schemas.QuotaWindow) -> QuotaWindow {
                QuotaWindow(days: Int(w.days), limit: Int(w.limit), used: Int(w.used), remaining: w.remaining.map(Int.init), restricted: w.restricted,
                            resetsAt: w.resetsAt.flatMap { $0 > 0 ? Date(timeIntervalSince1970: TimeInterval($0)) : nil })
            }
            return Quota(movie: window(body.movie), tv: window(body.tv))
        }
    }

    func permissionCatalog() async throws -> PermissionCatalog {
        try await load { client in
            guard case .ok(let ok) = try await client.getPermissions() else { throw APIError.unexpected }
            let body = try ok.body.json
            return PermissionCatalog(
                flags: body.flags.map { PermissionFlag(name: $0.name, value: $0.value, group: $0.group.rawValue, implies: $0.implies ?? [], reserved: $0.reserved) },
                defaults: PermissionDefaults(body.defaults)
            )
        }
    }

    func setPermissionDefaults(_ d: PermissionDefaults) async throws -> PermissionDefaults {
        try await load { client in
            let body = Components.Schemas.PermissionDefaults(
                movieQuotaDays: Int64(d.movieQuotaDays), movieQuotaLimit: Int64(d.movieQuotaLimit), permissions: d.permissions,
                tvQuotaDays: Int64(d.tvQuotaDays), tvQuotaLimit: Int64(d.tvQuotaLimit)
            )
            guard case .ok(let ok) = try await client.setPermissionDefaults(body: .json(body)) else { throw APIError.unexpected }
            return PermissionDefaults(try ok.body.json)
        }
    }

    /// Replaces a person's permission mask and request limits (a nil limit means the default applies).
    func setAccess(id: String, permissions: Int64, quota: UserQuota) async throws -> Profile {
        try await load { client in
            let q = Components.Schemas.UserQuota(movieDays: quota.movieDays.map(Int64.init), movieLimit: quota.movieLimit.map(Int64.init),
                                                 tvDays: quota.tvDays.map(Int64.init), tvLimit: quota.tvLimit.map(Int64.init))
            guard case .ok(let ok) = try await client.updateUser(path: .init(id: id), body: .json(.init(permissions: permissions, quota: q))) else { throw APIError.unexpected }
            return Profile(try ok.body.json)
        }
    }
}

extension PermissionDefaults {
    init(_ d: Components.Schemas.PermissionDefaults) {
        self.init(permissions: d.permissions, movieQuotaLimit: Int(d.movieQuotaLimit), movieQuotaDays: Int(d.movieQuotaDays),
                  tvQuotaLimit: Int(d.tvQuotaLimit), tvQuotaDays: Int(d.tvQuotaDays))
    }
}
