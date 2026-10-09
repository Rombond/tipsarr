import SwiftUI

/// Several accounts on one iPhone: switch, add, sign out of one.
struct AccountsView: View {
    @Environment(SessionManager.self) private var session
    @State private var removing: Account?

    var body: some View {
        List {
            Section {
                ForEach(session.accounts.accounts) { account in
                    Button { Task { await session.switchTo(account) } } label: {
                        HStack(spacing: Tokens.Spacing.md) {
                            AvatarView(userID: account.userID, name: account.name, size: 40)
                            VStack(alignment: .leading, spacing: 2) {
                                Text(verbatim: account.name).font(.body.weight(.medium))
                                Text(verbatim: account.serverURL.host() ?? account.serverURL.absoluteString)
                                    .font(.footnote).foregroundStyle(Tokens.palette.mutedFg)
                            }
                            Spacer()
                            if account.id == session.accounts.activeID {
                                Image(systemName: "checkmark").foregroundStyle(Tokens.palette.fg)
                                    .accessibilityLabel(Text("m.accounts.active"))
                            }
                        }
                        .frame(minHeight: Tokens.Size.touchTarget)
                        .contentShape(.rect)
                    }
                    .buttonStyle(.plain)
                    .swipeActions(edge: .trailing, allowsFullSwipe: false) {
                        Button(role: .destructive) { removing = account } label: {
                            Label("m.accounts.remove", systemImage: "rectangle.portrait.and.arrow.right")
                        }
                    }
                }
            } header: {
                Text("m.accounts.intro").textCase(nil)
            }
            Section {
                Button { session.beginAddAccount() } label: {
                    Label { Text("m.accounts.add") } icon: { Image(systemName: "plus.circle") }
                }
            } footer: {
                Text("m.accounts.footer_signout")
            }
        }
        .scrollContentBackground(.hidden)
        .background(Tokens.palette.bg)
        .navigationTitle("m.accounts.title")
        .navigationBarTitleDisplayMode(.inline)
        .confirmationDialog(
            Text("m.settings.sign_out_confirm"),
            isPresented: Binding(get: { removing != nil }, set: { if !$0 { removing = nil } }),
            titleVisibility: .visible, presenting: removing
        ) { account in
            Button("m.settings.sign_out", role: .destructive) { Task { await session.signOut(account) } }
            Button("common.cancel", role: .cancel) {}
        }
    }
}
