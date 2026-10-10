import SwiftUI

struct SettingsView: View {
    @Environment(SessionManager.self) private var session
    @Environment(AppSettings.self) private var settings
    @Environment(\.appContext) private var context
    @Environment(ToastCenter.self) private var toast
    @AppStorage("avatarVersion") private var avatarVersion = 0
    let account: Account
    let profile: Profile
    @State private var confirmSignOut = false
    @State private var showPicture = false
    @State private var deviceCount: Int?
    @State private var faceIDSignIn = false
    @State private var askFaceIDPassword = false
    @State private var faceIDPassword = ""
    @State private var push = PushManager.shared

    private var ratingSource: Binding<RatingSource> {
        Binding(
            get: { RatingSource(rawValue: profile.ratingSource) ?? .tmdb },
            set: { value in Task { await save(rating: value) } }
        )
    }

    private var region: Binding<String> {
        Binding(get: { profile.region.uppercased() }, set: { value in Task { await save(region: value) } })
    }

    /// The flag emoji of a 2-letter country code.
    static func flag(_ code: String) -> String {
        code.uppercased().unicodeScalars.compactMap { UnicodeScalar(127_397 + $0.value) }.map { String($0) }.joined()
    }

    /// Countries sorted by their name in the chosen language.
    private var regions: [(code: String, name: String)] {
        let locale = AppLanguage.locale
        return Locale.Region.isoRegions
            .filter { $0.identifier.count == 2 && $0.subRegions.isEmpty }
            .compactMap { region in locale.localizedString(forRegionCode: region.identifier).map { (region.identifier, $0) } }
            .sorted { $0.name.localizedCaseInsensitiveCompare($1.name) == .orderedAscending }
    }

    var body: some View {
        @Bindable var settings = settings
        List {
            Section("m.settings.account") {
                ProfileLink(route: .accounts, chevron: true) {
                    LabeledContent {
                        Text(verbatim: String(session.accounts.accounts.count))
                    } label: {
                        Label { Text(session.accounts.accounts.count == 1 ? "m.accounts.title_one" : "m.accounts.title") } icon: { Image(systemName: "person.2") }
                    }
                }
                Button { showPicture = true } label: {
                    HStack {
                        Label { Text("profile.change_picture") } icon: { Image(systemName: "person.crop.circle") }
                        Spacer()
                        Image(systemName: "chevron.right").font(.footnote.weight(.semibold)).foregroundStyle(Tokens.palette.mutedFg)
                    }
                    .contentShape(.rect)
                }
                .buttonStyle(.plain)
                NavigationLink {
                    RegionPickerScreen(regions: regions, selection: region)
                } label: {
                    LabeledContent {
                        Text(verbatim: regions.first { $0.code == profile.region.uppercased() }.map { "\(Self.flag($0.code)) \($0.name)" } ?? "")
                    } label: {
                        Label { Text("m.settings.region") } icon: { Image(systemName: "globe") }
                    }
                }
                NavigationLink {
                    RatingSourceScreen(selection: ratingSource)
                } label: {
                    LabeledContent {
                        HStack(spacing: Tokens.Spacing.sm) {
                            ProviderMark(source: ratingSource.wrappedValue, value: 80, height: 14)
                            Text(ratingSource.wrappedValue.title)
                        }
                    } label: {
                        Label { Text("profile.rating_source") } icon: { Image(systemName: "star") }
                    }
                }
                ProfileLink(route: .hidden, chevron: true) {
                    Label { Text("m.settings.hidden") } icon: { Image(systemName: "eye.slash") }
                }
            }
            Section("m.settings.appearance") {
                Picker(selection: $settings.theme) {
                    ForEach(AppTheme.allCases, id: \.self) { theme in
                        Label { Text(theme.title) } icon: { Image(systemName: theme.symbol) }.tag(theme)
                    }
                } label: {
                    Label { Text("theme.title") } icon: { Image(systemName: "paintpalette") }
                }
                .modifier(AdaptivePickerStyle())
                ProfileLink(route: .appIcon, chevron: true) {
                    Label { Text("m.settings.app_icon") } icon: { Image(systemName: "app.badge") }
                }
            }
            if session.serverStatus?.pushAvailable == true {
                notificationsSection
            }
            Section {
                if DeviceAuth.isAvailable {
                    Toggle(isOn: Binding(get: { faceIDSignIn }, set: { on in
                        if on { faceIDPassword = ""; askFaceIDPassword = true } else { CredentialStore.delete(accountID: account.id); faceIDSignIn = false }
                    })) {
                        Label { Text("m.settings.faceid_signin") } icon: { Image(systemName: "key.viewfinder") }
                    }
                }
                ProfileLink(route: .devices, chevron: true) {
                    LabeledContent {
                        if let deviceCount { Text(verbatim: String(deviceCount)) }
                    } label: {
                        Label { Text("profile.devices_title") } icon: { Image(systemName: "macbook.and.iphone") }
                    }
                }
            } header: {
                Text("m.settings.security")
            } footer: {
                if DeviceAuth.isAvailable { Text("m.settings.faceid_signin_footer") }
            }
            Section("m.settings.server") {
                Label {
                    VStack(alignment: .leading, spacing: 2) {
                        Text(verbatim: account.serverURL.host() ?? account.serverURL.absoluteString)
                        if let version = session.serverStatus?.version {
                            Text(verbatim: L10n.string("m.settings.server_version", version))
                                .font(.footnote).foregroundStyle(Tokens.palette.mutedFg)
                        }
                    }
                } icon: { Image(systemName: "server.rack") }
            }
            Section {
                Button(role: .destructive) { confirmSignOut = true } label: {
                    // Full row width: the dialog points at the middle of the row, right above the text.
                    Label { Text("m.settings.sign_out") } icon: { Image(systemName: "rectangle.portrait.and.arrow.right") }
                        .frame(maxWidth: .infinity, alignment: .center)
                }
                // On the button itself, so the dialog appears next to it.
                .confirmationDialog(Text("m.settings.sign_out_confirm"), isPresented: $confirmSignOut, titleVisibility: .visible) {
                    Button("m.settings.sign_out", role: .destructive) { Task { await session.signOut(account) } }
                    Button("common.cancel", role: .cancel) {}
                }
            } footer: {
                Text("m.accounts.footer_signout")
            }
        }
        .scrollContentBackground(.hidden)
        .readableColumn()
        .background(Tokens.palette.bg)
        .navigationTitle("m.settings.title")
        .navigationBarTitleDisplayMode(.inline)
        .task {
            deviceCount = try? await context?.api.sessions().count
            faceIDSignIn = CredentialStore.exists(accountID: account.id)
        }
        .alert(Text("m.settings.faceid_password_title"), isPresented: $askFaceIDPassword) {
            SecureField("m.settings.faceid_password_field", text: $faceIDPassword)
            Button("m.settings.faceid_enable") { Task { await enableFaceID() } }
            Button("common.cancel", role: .cancel) {}
        } message: {
            Text("m.settings.faceid_password_message")
        }
        .sheet(isPresented: $showPicture) {
            PictureSheet(profile: profile) { avatarVersion += 1 }
                .tipsarrSheet(detents: [.medium])
        }
    }

    // MARK: Notifications

    private var notificationsSection: some View {
        Section {
            Toggle(isOn: Binding(get: { push.isOn }, set: { on in Task { if on { await push.turnOn() } else { await push.turnOff() } } })) {
                Label { Text("m.settings.notifications_toggle") } icon: { Image(systemName: "bell.badge") }
            }
            .disabled(push.isDenied)
            if push.isDenied {
                Button { push.openSystemSettings() } label: {
                    Label { Text("m.settings.notifications_open_settings") } icon: { Image(systemName: "gearshape") }
                }
            }
            if push.isOn {
                categoryToggle(.requests, "m.settings.notifications_requests", symbol: "film.stack")
                if profile.isAdmin { categoryToggle(.admin, "m.settings.notifications_admin", symbol: "checkmark.seal") }
                categoryToggle(.issues, "m.settings.notifications_issues", symbol: "exclamationmark.bubble")
            }
        } header: {
            Text("m.settings.notifications")
        } footer: {
            Text(push.isDenied ? "m.settings.notifications_denied" : "m.settings.notifications_footer")
        }
    }

    private func categoryToggle(_ category: PushManager.Category, _ title: LocalizedStringKey, symbol: String) -> some View {
        Toggle(isOn: Binding(
            get: { push.categories & category.rawValue != 0 },
            set: { on in Task { await push.set(category, on: on) } }
        )) {
            Label { Text(title) } icon: { Image(systemName: symbol) }
        }
    }

    // MARK: Saving

    private func enableFaceID() async {
        let password = faceIDPassword
        faceIDPassword = ""
        do {
            try await session.rememberLogin(for: account, password: password)
            faceIDSignIn = true
        } catch {
            toast.show(L10n.string("m.settings.faceid_failed"), kind: .error)
        }
    }

    private func save(rating: RatingSource) async {
        guard let context else { return }
        do {
            session.update(profile: try await context.api.updatePreferences(ratingSource: rating))
            toast.show(L10n.string("profile.saved"))
        } catch {
            toast.show(APIError.from(error).localizedMessage, kind: .error)
        }
    }

    private func save(region code: String) async {
        guard let context else { return }
        do {
            session.update(profile: try await context.api.updatePreferences(region: code))
            toast.show(L10n.string("profile.saved"))
        } catch {
            toast.show(APIError.from(error).localizedMessage, kind: .error)
        }
    }
}

/// Countries with a search field: about 250 entries are too many for a plain list.
private struct RegionPickerScreen: View {
    let regions: [(code: String, name: String)]
    @Binding var selection: String
    @Environment(\.dismiss) private var dismiss
    @State private var query = ""

    private var shown: [(code: String, name: String)] {
        query.isEmpty ? regions : regions.filter { $0.name.localizedStandardContains(query) || $0.code.localizedCaseInsensitiveContains(query) }
    }

    var body: some View {
        List(shown, id: \.code) { region in
            Button {
                selection = region.code
                dismiss()
            } label: {
                HStack {
                    Text(verbatim: SettingsView.flag(region.code)).font(.title3)
                    Text(verbatim: region.name).foregroundStyle(Tokens.palette.fg)
                    Spacer()
                    if region.code == selection { Image(systemName: "checkmark").foregroundStyle(Tokens.palette.fg) }
                }
            }
        }
        .scrollContentBackground(.hidden)
        .background(Tokens.palette.bg)
        .searchable(text: $query, placement: .navigationBarDrawer(displayMode: .always), prompt: Text("m.settings.region"))
        .navigationTitle("m.settings.region")
        .navigationBarTitleDisplayMode(.inline)
    }
}

/// The score shown on posters: each choice with its logo.
private struct RatingSourceScreen: View {
    @Binding var selection: RatingSource
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        List(RatingSource.allCases, id: \.self) { source in
            Button {
                selection = source
                dismiss()
            } label: {
                HStack(spacing: Tokens.Spacing.md) {
                    ProviderMark(source: source, value: 80, height: 18)
                        .frame(width: 56, alignment: .leading)
                    Text(source.title).foregroundStyle(Tokens.palette.fg)
                    Spacer()
                    if source == selection { Image(systemName: "checkmark").foregroundStyle(Tokens.palette.fg) }
                }
                .frame(minHeight: Tokens.Size.touchTarget)
                .contentShape(.rect)
            }
        }
        .scrollContentBackground(.hidden)
        .background(Tokens.palette.bg)
        .navigationTitle("profile.rating_source")
        .navigationBarTitleDisplayMode(.inline)
    }
}

private extension RatingSource {
    var title: LText {
        switch self {
        case .tmdb: "m.settings.rating_tmdb"
        case .imdb: "m.settings.rating_imdb"
        case .metacritic: "m.settings.rating_metacritic"
        case .rottenTomatoes: "m.settings.rating_rt"
        }
    }
}
