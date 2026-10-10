import SwiftUI

/// The checkbox list of permission flags. Like in Seerr, a flag that brings others with it (manage requests brings
/// auto-approve, request brings movies and TV shows ...) switches them on and keeps them on while it is on.
struct PermissionEditor: View {
    let catalog: PermissionCatalog
    @Binding var mask: Int64
    /// Read-only (an admin, or your own account).
    var readOnly = false
    /// The keys to users and settings can only be given by admins.
    var keysLocked = false

    private let groups = ["requests", "auto-approve", "issues", "admin"]

    var body: some View {
        let locked = catalog.forced(mask)
        ForEach(groups, id: \.self) { group in
            let flags = catalog.flags.filter { $0.group == group && !$0.reserved }
            if !flags.isEmpty {
                Section {
                    ForEach(flags) { flag in
                        let isKey = flag.value == Permission.manageUsers.rawValue || flag.value == Permission.manageSettings.rawValue
                        let off = readOnly || (locked & flag.value != 0 && mask & flag.value != 0) || (keysLocked && isKey)
                        Toggle(isOn: Binding(
                            get: { mask & flag.value != 0 },
                            set: { on in mask = catalog.normalize(on ? mask | flag.value : mask & ~flag.value) }
                        )) {
                            VStack(alignment: .leading, spacing: 2) {
                                Text(verbatim: L10n.string("perm.\(flag.name)")).font(.body)
                                Text(verbatim: L10n.string("perm.\(flag.name).desc")).font(.footnote).foregroundStyle(Tokens.palette.mutedFg)
                            }
                        }
                        .disabled(off)
                    }
                } header: {
                    Text(verbatim: L10n.string("perm.group.\(group)"))
                }
            }
        }
    }
}

/// Request limits: how many movies and seasons in how many days. Empty = the default, 0 = no limit.
struct QuotaFields: View {
    @Binding var movieLimit: String
    @Binding var movieDays: String
    @Binding var tvLimit: String
    @Binding var tvDays: String
    /// Shown as the placeholder: what applies when a field is empty.
    var defaults: PermissionDefaults?
    var readOnly = false

    var body: some View {
        Section {
            row("users.quota_movies", $movieLimit, $movieDays, defaults.map { ($0.movieQuotaLimit, $0.movieQuotaDays) })
            row("users.quota_shows", $tvLimit, $tvDays, defaults.map { ($0.tvQuotaLimit, $0.tvQuotaDays) })
        } header: {
            Text("users.quota_title")
        } footer: {
            Text("users.quota_hint")
        }
    }

    private func row(_ title: LText, _ limit: Binding<String>, _ days: Binding<String>, _ placeholder: (Int, Int)?) -> some View {
        HStack(spacing: Tokens.Spacing.md) {
            Text(title)
            Spacer(minLength: 0)
            field(L10n.string("users.quota_limit"), limit, placeholder.map { String($0.0) })
            field(L10n.string("users.quota_days"), days, placeholder.map { String($0.1) })
        }
        .disabled(readOnly)
    }

    private func field(_ label: String, _ text: Binding<String>, _ placeholder: String?) -> some View {
        TextField(placeholder ?? label, text: text)
            .keyboardType(.numberPad)
            .multilineTextAlignment(.trailing)
            .frame(width: 64)
            .accessibilityLabel(label)
    }
}

private func quotaText(_ n: Int?) -> String { n.map(String.init) ?? "" }
private func quotaNumber(_ s: String) -> Int? { Int(s.trimmingCharacters(in: .whitespaces)) }

/// Permissions and request limits of one person.
struct UserAccessSheet: View {
    let user: Profile
    let isMe: Bool
    var onSaved: (Profile) -> Void
    @Environment(\.appContext) private var context
    @Environment(\.dismiss) private var dismiss
    @Environment(ToastCenter.self) private var toast
    @State private var catalog: PermissionCatalog?
    @State private var mask: Int64 = 0
    @State private var movieLimit = ""
    @State private var movieDays = ""
    @State private var tvLimit = ""
    @State private var tvDays = ""
    @State private var saving = false
    @State private var error: APIError?

    private var readOnly: Bool { user.isAdmin || isMe }

    var body: some View {
        NavigationStack {
            Group {
                if let catalog {
                    List {
                        if user.isAdmin { Section { Text("users.permissions_admin").font(.footnote) } }
                        else if isMe { Section { Text("users.permissions_own").font(.footnote) } }
                        else if context?.profile.isAdmin != true { Section { Text("users.permissions_keys").font(.footnote) } }
                        PermissionEditor(catalog: catalog, mask: $mask, readOnly: readOnly, keysLocked: context?.profile.isAdmin != true)
                        QuotaFields(movieLimit: $movieLimit, movieDays: $movieDays, tvLimit: $tvLimit, tvDays: $tvDays, defaults: catalog.defaults, readOnly: readOnly)
                    }
                } else if let error {
                    ErrorState(error: error) { await load() }
                } else {
                    ProgressView().frame(maxWidth: .infinity, maxHeight: .infinity)
                }
            }
            .scrollContentBackground(.hidden)
            .background(Tokens.palette.bg)
            .navigationTitle(Text(verbatim: L10n.string("users.permissions_title", user.name)))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) { Button("common.cancel") { dismiss() } }
                ToolbarItem(placement: .confirmationAction) {
                    Button(saving ? "common.saving" : "common.save") { Task { await save() } }
                        .disabled(saving || readOnly || catalog == nil)
                }
            }
        }
        .task { await load() }
    }

    private func load() async {
        guard let context else { return }
        do {
            catalog = try await context.api.permissionCatalog()
            error = nil
            mask = user.permissions
            movieLimit = quotaText(user.quota.movieLimit)
            movieDays = quotaText(user.quota.movieDays)
            tvLimit = quotaText(user.quota.tvLimit)
            tvDays = quotaText(user.quota.tvDays)
        } catch {
            self.error = APIError.from(error)
        }
    }

    private func save() async {
        guard let context else { return }
        saving = true
        defer { saving = false }
        do {
            let quota = UserQuota(movieLimit: quotaNumber(movieLimit), movieDays: quotaNumber(movieDays), tvLimit: quotaNumber(tvLimit), tvDays: quotaNumber(tvDays))
            let updated = try await context.api.setAccess(id: user.id, permissions: mask, quota: quota)
            toast.show(L10n.string("users.permissions_saved"))
            onSaved(updated)
            dismiss()
        } catch {
            toast.show(APIError.from(error).localizedMessage, kind: .error)
        }
    }
}

/// What people get when they sign in for the first time. Only admins can change it.
struct PermissionDefaultsView: View {
    @Environment(\.appContext) private var context
    @Environment(ToastCenter.self) private var toast
    @State private var catalog: PermissionCatalog?
    @State private var mask: Int64 = 0
    @State private var movieLimit = ""
    @State private var movieDays = ""
    @State private var tvLimit = ""
    @State private var tvDays = ""
    @State private var saving = false
    @State private var error: APIError?

    private var isAdmin: Bool { context?.profile.isAdmin == true }

    var body: some View {
        Group {
            if let catalog {
                List {
                    Section { Text("users.defaults_desc").font(.footnote).foregroundStyle(Tokens.palette.mutedFg) }
                    PermissionEditor(catalog: catalog, mask: $mask, readOnly: !isAdmin)
                    QuotaFields(movieLimit: $movieLimit, movieDays: $movieDays, tvLimit: $tvLimit, tvDays: $tvDays, readOnly: !isAdmin)
                }
            } else if let error {
                ErrorState(error: error) { await load() }
            } else {
                ProgressView().frame(maxWidth: .infinity, maxHeight: .infinity)
            }
        }
        .scrollContentBackground(.hidden)
        .readableColumn()
        .background(Tokens.palette.bg)
        .navigationTitle("users.defaults_title")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            if isAdmin {
                ToolbarItem(placement: .confirmationAction) {
                    Button(saving ? "common.saving" : "common.save") { Task { await save() } }.disabled(saving || catalog == nil)
                }
            }
        }
        .task { await load() }
    }

    private func load() async {
        guard let context else { return }
        do {
            let c = try await context.api.permissionCatalog()
            catalog = c
            error = nil
            mask = c.defaults.permissions
            movieLimit = String(c.defaults.movieQuotaLimit)
            movieDays = String(c.defaults.movieQuotaDays)
            tvLimit = String(c.defaults.tvQuotaLimit)
            tvDays = String(c.defaults.tvQuotaDays)
        } catch {
            self.error = APIError.from(error)
        }
    }

    private func save() async {
        guard let context else { return }
        saving = true
        defer { saving = false }
        do {
            _ = try await context.api.setPermissionDefaults(PermissionDefaults(
                permissions: mask, movieQuotaLimit: quotaNumber(movieLimit) ?? 0, movieQuotaDays: quotaNumber(movieDays) ?? 0,
                tvQuotaLimit: quotaNumber(tvLimit) ?? 0, tvQuotaDays: quotaNumber(tvDays) ?? 0))
            toast.show(L10n.string("common.saved"))
        } catch {
            toast.show(APIError.from(error).localizedMessage, kind: .error)
        }
    }
}
