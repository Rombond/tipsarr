import SwiftUI

struct UsersView: View {
    @Environment(\.appContext) private var context
    @State private var users: [Profile] = []
    @State private var phase: Phase = .loading

    enum Phase: Equatable { case loading, loaded, failed(APIError) }

    var body: some View {
        List {
            Section {
                switch phase {
                case .loading:
                    ProgressView().frame(maxWidth: .infinity).listRowBackground(Color.clear)
                case .failed(let error):
                    ErrorState(error: error) { await load() }.listRowBackground(Color.clear)
                case .loaded:
                    ForEach(users, id: \.id) { user in
                        NavigationLink(value: ProfileRoute.user(user.id)) { row(user) }
                    }
                }
            } header: {
                Text("users.intro").textCase(nil).font(.footnote)
            }
        }
        .scrollContentBackground(.hidden)
        .background(Tokens.palette.bg)
        .navigationTitle("users.title")
        .navigationBarTitleDisplayMode(.inline)
        .task { await load() }
        .refreshable { await load() }
    }

    private func row(_ user: Profile) -> some View {
        HStack(spacing: Tokens.Spacing.md) {
            AvatarView(userID: user.id, name: user.name, size: 40)
            VStack(alignment: .leading, spacing: 2) {
                HStack(spacing: Tokens.Spacing.sm) {
                    Text(verbatim: user.name).font(.body.weight(.medium))
                    if user.isAdmin {
                        Text("users.role_admin").font(.caption2.weight(.semibold))
                            .padding(.horizontal, Tokens.Spacing.sm).padding(.vertical, 2)
                            .background(Tokens.palette.muted, in: .capsule)
                    }
                }
                if let seen = user.lastLoginAt, seen.timeIntervalSince1970 > 0 {
                    Text(verbatim: L10n.string("users.last_signin", seen.formatted(.relative(presentation: .named))))
                        .font(.footnote).foregroundStyle(Tokens.palette.mutedFg)
                }
            }
        }
        .accessibilityElement(children: .combine)
    }

    private func load() async {
        guard let context else { return }
        do {
            users = try await context.api.users().sorted { ($0.lastLoginAt ?? .distantPast) > ($1.lastLoginAt ?? .distantPast) }
            phase = .loaded
        } catch {
            if users.isEmpty { phase = .failed(APIError.from(error)) }
        }
    }
}

struct UserDetailView: View {
    let id: String
    @Environment(\.appContext) private var context
    @Environment(ToastCenter.self) private var toast
    @State private var detail: UserDetail?
    @State private var error: APIError?

    private var isMe: Bool { context?.profile.id == id }

    var body: some View {
        Group {
            if let detail { content(detail) } else if let error {
                ErrorState(error: error) { await load() }
            } else {
                ProgressView().frame(maxWidth: .infinity, maxHeight: .infinity)
            }
        }
        .background(Tokens.palette.bg)
        .navigationTitle(Text(verbatim: detail?.profile.name ?? ""))
        .navigationBarTitleDisplayMode(.inline)
        .task { await load() }
    }

    private func content(_ detail: UserDetail) -> some View {
        let user = detail.profile
        return ScrollView {
            VStack(spacing: Tokens.Spacing._2xl) {
                VStack(spacing: Tokens.Spacing.sm) {
                    AvatarView(userID: user.id, name: user.name, size: 88)
                    Text(verbatim: user.name).font(.title.weight(.bold))
                    if let created = user.createdAt, created.timeIntervalSince1970 > 0 {
                        Text(verbatim: L10n.string("profile.member_since", created.formatted(.dateTime.month(.abbreviated).year())))
                            .font(.footnote).foregroundStyle(Tokens.palette.mutedFg)
                    }
                }
                VStack(alignment: .leading, spacing: Tokens.Spacing.sm) {
                    Text("users.role").font(.footnote.weight(.semibold)).foregroundStyle(Tokens.palette.mutedFg)
                    Picker("users.role", selection: Binding(get: { user.isAdmin }, set: { admin in Task { await setRole(admin) } })) {
                        Text("users.role_user").tag(false)
                        Text("users.role_admin").tag(true)
                    }
                    .pickerStyle(.segmented)
                    .disabled(isMe)
                    if isMe { Text("m.users.own_role").font(.footnote).foregroundStyle(Tokens.palette.mutedFg) }
                }
                stats(detail.stats)
            }
            .padding(Tokens.Spacing.lg)
            .frame(maxWidth: 720)
            .frame(maxWidth: .infinity)
        }
    }

    private func stats(_ s: UserStats) -> some View {
        VStack(alignment: .leading, spacing: Tokens.Spacing.md) {
            Text("m.users.stats").font(.title3.weight(.bold)).accessibilityAddTraits(.isHeader)
            LazyVGrid(columns: [GridItem(.adaptive(minimum: 100), spacing: Tokens.Spacing.md)], spacing: Tokens.Spacing.md) {
                tile("m.users.requests", s.requests)
                tile("m.users.pending", s.pending)
                tile("m.users.approved", s.approved)
                tile("m.users.available", s.available)
                tile("m.users.declined", s.declined)
                tile("m.users.failed", s.failed)
                tile("m.users.movies", s.movies)
                tile("m.users.shows", s.shows)
                tile("m.users.watchlist", s.watchlist)
                tile("m.users.watched", s.watched)
            }
        }
    }

    private func tile(_ label: LText, _ value: Int) -> some View {
        VStack(spacing: Tokens.Spacing.xs) {
            Text(verbatim: String(value)).font(.title2.weight(.bold))
            Text(label).font(.caption).foregroundStyle(Tokens.palette.mutedFg).multilineTextAlignment(.center)
        }
        .frame(maxWidth: .infinity)
        .padding(.vertical, Tokens.Spacing.md)
        .background(Tokens.palette.card, in: .rect(cornerRadius: Tokens.Radius.md))
        .overlay { RoundedRectangle(cornerRadius: Tokens.Radius.md).strokeBorder(Tokens.palette.border) }
        .accessibilityElement(children: .combine)
    }

    private func load() async {
        guard let context else { return }
        do {
            detail = try await context.api.userDetail(id: id)
            error = nil
        } catch {
            if detail == nil { self.error = APIError.from(error) }
        }
    }

    private func setRole(_ admin: Bool) async {
        guard let context, var current = detail else { return }
        do {
            let updated = try await context.api.setRole(id: id, admin: admin)
            current.profile.role = updated.role
            detail = current
            toast.show(L10n.string("m.users.saved"))
        } catch {
            toast.show(APIError.from(error).localizedMessage, kind: .error)
        }
    }
}
