import SwiftUI

struct SettingsView: View {
    @Environment(SessionManager.self) private var session
    @Environment(AppSettings.self) private var settings
    @Environment(\.appContext) private var context
    @Environment(ToastCenter.self) private var toast
    let account: Account
    let profile: Profile
    @State private var confirmSignOut = false

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

    var body: some View {
        @Bindable var settings = settings
        List {
            Section("m.settings.appearance") {
                Picker(selection: $settings.theme) {
                    ForEach(AppTheme.allCases, id: \.self) { theme in
                        Label { Text(theme.title) } icon: { Image(systemName: theme.symbol) }.tag(theme)
                    }
                } label: {
                    Label { Text("theme.title") } icon: { Image(systemName: "circle.lefthalf.filled") }
                }
                .pickerStyle(.navigationLink)
                Picker(selection: language) {
                    Text("m.settings.language_default").tag(String?.none)
                    ForEach(AppLanguage.supported, id: \.code) { item in Text(verbatim: item.name).tag(String?.some(item.code)) }
                } label: {
                    Label { Text("lang.title") } icon: { Image(systemName: "globe") }
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
            }
            Section("m.settings.account") {
                NavigationLink(value: ProfileRoute.accounts) {
                    Label { Text("m.accounts.manage") } icon: { Image(systemName: "person.2") }
                }
                NavigationLink(value: ProfileRoute.devices) {
                    Label { Text("profile.devices_title") } icon: { Image(systemName: "iphone.and.ipad") }
                }
            }
            Section {
                Button(role: .destructive) { confirmSignOut = true } label: {
                    Label { Text("m.settings.sign_out") } icon: { Image(systemName: "rectangle.portrait.and.arrow.right") }
                }
            } footer: {
                Text("m.accounts.footer_signout")
            }
            Section("m.profile.about") {
                LabeledContent("m.profile.version") { Text(verbatim: TipsarrClient.appVersion) }
            }
        }
        .scrollContentBackground(.hidden)
        .background(Tokens.palette.bg)
        .navigationTitle("m.settings.title")
        .navigationBarTitleDisplayMode(.inline)
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
