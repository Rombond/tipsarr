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

    private var ratingSource: Binding<RatingSource> {
        Binding(
            get: { RatingSource(rawValue: profile.ratingSource) ?? .tmdb },
            set: { value in Task { await save(rating: value) } }
        )
    }

    /// `nil` follows the iPhone.
    private var language: Binding<String?> {
        Binding(get: { settings.language }, set: { value in Task { await save(language: value) } })
    }

    private var region: Binding<String> {
        Binding(get: { profile.region.uppercased() }, set: { value in Task { await save(region: value) } })
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
                NavigationLink(value: ProfileRoute.accounts) {
                    LabeledContent {
                        Text(verbatim: String(session.accounts.accounts.count))
                    } label: {
                        Label { Text("m.accounts.title") } icon: { Image(systemName: "person.2") }
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
                Picker(selection: language) {
                    Text("m.settings.language_default").tag(String?.none)
                    ForEach(AppLanguage.supported, id: \.code) { item in Text(verbatim: item.name).tag(String?.some(item.code)) }
                } label: {
                    Label { Text("lang.title") } icon: { Image(systemName: "character.bubble") }
                }
                .pickerStyle(.navigationLink)
                Picker(selection: region) {
                    ForEach(regions, id: \.code) { item in Text(verbatim: item.name).tag(item.code) }
                } label: {
                    Label { Text("m.settings.region") } icon: { Image(systemName: "globe") }
                }
                .pickerStyle(.navigationLink)
                Picker(selection: ratingSource) {
                    Text("m.settings.rating_tmdb").tag(RatingSource.tmdb)
                    Text("m.settings.rating_imdb").tag(RatingSource.imdb)
                    Text("m.settings.rating_metacritic").tag(RatingSource.metacritic)
                    Text("m.settings.rating_rt").tag(RatingSource.rottenTomatoes)
                } label: {
                    Label { Text("profile.rating_source") } icon: { Image(systemName: "star") }
                }
                .pickerStyle(.navigationLink)
                NavigationLink(value: ProfileRoute.hidden) {
                    Label { Text("profile.hidden_title") } icon: { Image(systemName: "eye.slash") }
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
                .pickerStyle(.navigationLink)
                NavigationLink(value: ProfileRoute.appIcon) {
                    Label { Text("m.settings.app_icon") } icon: { Image(systemName: "app.badge") }
                }
            }
            Section("m.settings.security") {
                NavigationLink(value: ProfileRoute.devices) {
                    LabeledContent {
                        if let deviceCount { Text(verbatim: String(deviceCount)) }
                    } label: {
                        Label { Text("profile.devices_title") } icon: { Image(systemName: "macbook.and.iphone") }
                    }
                }
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
                    Label { Text("m.settings.sign_out") } icon: { Image(systemName: "rectangle.portrait.and.arrow.right") }
                }
            } footer: {
                Text("m.accounts.footer_signout")
            }
        }
        .scrollContentBackground(.hidden)
        .background(Tokens.palette.bg)
        .navigationTitle("m.settings.title")
        .navigationBarTitleDisplayMode(.inline)
        .task { deviceCount = try? await context?.api.sessions().count }
        .sheet(isPresented: $showPicture) {
            PictureSheet(profile: profile) { avatarVersion += 1 }
                .tipsarrSheet(detents: [.medium])
        }
        .confirmationDialog(Text("m.settings.sign_out_confirm"), isPresented: $confirmSignOut, titleVisibility: .visible) {
            Button("m.settings.sign_out", role: .destructive) { Task { await session.signOut(account) } }
            Button("common.cancel", role: .cancel) {}
        }
    }

    // MARK: Saving

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

    /// The interface changes at once; the server learns it too so titles and descriptions follow.
    private func save(language code: String?) async {
        settings.language = code
        guard let context else { return }
        do {
            session.update(profile: try await context.api.updatePreferences(language: code ?? ""))
            toast.show(L10n.string("profile.saved_titles"))
        } catch {
            toast.show(APIError.from(error).localizedMessage, kind: .error)
        }
    }
}
