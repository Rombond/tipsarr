import SwiftUI

struct LoginView: View {
    @Environment(SessionManager.self) private var session
    let server: Server
    @State var username: String
    @State private var password = ""
    @State private var signingIn = false
    @State private var errorText: String?
    @FocusState private var focus: Focus?

    private enum Focus { case username, password }

    init(server: Server, prefillUsername: String?) {
        self.server = server
        _username = State(initialValue: prefillUsername ?? "")
    }

    var body: some View {
        AuthScaffold(symbol: "person.crop.circle", title: "login.title", subtitle: "m.login.subtitle") {
            VStack(alignment: .leading, spacing: Tokens.Spacing.lg) {
                Label { Text(verbatim: server.host) } icon: { Image(systemName: "server.rack") }
                    .font(.footnote)
                    .foregroundStyle(Tokens.palette.mutedFg)
                Field(title: "login.username", text: $username, contentType: .username, submitLabel: .next)
                    .focused($focus, equals: .username)
                    .onSubmit { focus = .password }
                Field(title: "login.password", text: $password, secure: true, contentType: .password, submitLabel: .go)
                    .focused($focus, equals: .password)
                    .onSubmit { Task { await submit() } }
                if let errorText {
                    Banner(kind: .error, title: LText.verbatim(errorText))
                }
                Button { Task { await submit() } } label: {
                    if signingIn {
                        HStack(spacing: Tokens.Spacing.sm) { ProgressView(); Text("m.login.signing_in") }
                    } else {
                        Text("login.submit")
                    }
                }
                .buttonStyle(.tipsarr(.primary, fullWidth: true))
                .disabled(signingIn || username.isEmpty || password.isEmpty)
                HStack {
                    Button("m.accounts.other_server") { session.backToConnect() }
                    if session.accounts.active != nil {
                        Spacer()
                        Button("common.cancel") { Task { await session.cancelAddAccount() } }
                    }
                }
                .buttonStyle(.tipsarr(.ghost))
            }
        }
        .onAppear { focus = username.isEmpty ? .username : .password }
    }

    private func submit() async {
        guard !signingIn else { return }
        signingIn = true
        errorText = nil
        defer { signingIn = false }
        do {
            try await session.signIn(server: server, username: username, password: password)
        } catch {
            password = ""
            errorText = APIError.from(error).localizedMessage
        }
    }
}
