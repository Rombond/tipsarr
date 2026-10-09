import SwiftUI

struct ConnectView: View {
    @Environment(SessionManager.self) private var session
    @State private var address = ""
    @State private var checking = false
    @State private var error: LText?

    var body: some View {
        AuthScaffold(symbol: "server.rack", title: "m.connect.title", subtitle: "m.connect.subtitle") {
            VStack(alignment: .leading, spacing: Tokens.Spacing.lg) {
                Field(title: "m.connect.address", text: $address, prompt: "m.connect.placeholder",
                      error: error, keyboard: .URL, contentType: .URL, submitLabel: .go)
                    .onSubmit { Task { await submit() } }
                    .onChange(of: address) { error = nil }
                Button { Task { await submit() } } label: {
                    if checking {
                        HStack(spacing: Tokens.Spacing.sm) { ProgressView(); Text("m.connect.checking") }
                    } else {
                        Text("m.connect.continue")
                    }
                }
                .buttonStyle(.tipsarr(.primary, fullWidth: true))
                .disabled(checking || address.trimmingCharacters(in: .whitespaces).isEmpty)
                if session.accounts.active != nil {
                    Button("common.cancel") { Task { await session.cancelAddAccount() } }
                        .buttonStyle(.tipsarr(.ghost, fullWidth: true))
                }
                Text("m.connect.footer").font(.footnote).foregroundStyle(Tokens.palette.mutedFg)
            }
        }
    }

    private func submit() async {
        guard !checking else { return }
        guard let url = ServerAddress.normalize(address) else {
            error = "error.url_invalid"
            return
        }
        checking = true
        defer { checking = false }
        do {
            let server = try await session.connect(to: url)
            session.continueToLogin(server)
        } catch let failure as APIError {
            // Reachable but not ours (404, HTML, bad JSON) vs not reachable at all.
            error = failure == .unreachable ? "m.connect.unreachable" : "m.connect.not_tipsarr"
        } catch {
            self.error = "m.connect.unreachable"
        }
    }
}
