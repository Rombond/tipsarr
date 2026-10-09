import SwiftUI

/// The tab bar of a landscape iPad window: a sidebar with the app name on top and the account at the bottom.
struct AppSidebar: View {
    @Binding var selection: AppTab
    let account: Account
    let profile: Profile
    var badges: [AppTab: Int] = [:]

    var body: some View {
        VStack(alignment: .leading, spacing: Tokens.Spacing.xs) {
            HStack(spacing: Tokens.Spacing.md) {
                Image("IconPreviewDefault")
                    .resizable()
                    .frame(width: 36, height: 36)
                    .clipShape(.rect(cornerRadius: 8, style: .continuous))
                    .accessibilityHidden(true)
                Text("Tipsarr").font(.title2.weight(.bold))
            }
            .padding(.horizontal, Tokens.Spacing.md)
            .padding(.top, Tokens.Spacing.xl)
            .padding(.bottom, Tokens.Spacing._2xl)
            ForEach(AppTab.ordered, id: \.self) { tab in
                Button { selection = tab } label: {
                    HStack(spacing: Tokens.Spacing.md) {
                        Image(systemName: tab.symbol).frame(width: 24)
                        Text(tab.title).font(.body.weight(selection == tab ? .semibold : .medium))
                        Spacer(minLength: 0)
                        CountBadge(count: badges[tab] ?? 0)
                    }
                    .foregroundStyle(selection == tab ? Tokens.palette.fg : Tokens.palette.mutedFg)
                    .padding(.horizontal, Tokens.Spacing.md)
                    .frame(minHeight: Tokens.Size.touchTarget)
                    .background { if selection == tab { RoundedRectangle(cornerRadius: Tokens.Radius.md).fill(Tokens.palette.muted) } }
                    .contentShape(.rect(cornerRadius: Tokens.Radius.md))
                }
                .buttonStyle(.plain)
                .accessibilityAddTraits(selection == tab ? .isSelected : [])
            }
            Spacer(minLength: 0)
            Button { selection = .profile } label: {
                HStack(spacing: Tokens.Spacing.md) {
                    AvatarView(userID: profile.id, name: profile.name, size: 36)
                    VStack(alignment: .leading, spacing: 2) {
                        Text(verbatim: profile.name).font(.subheadline.weight(.semibold)).foregroundStyle(Tokens.palette.fg)
                        Text(verbatim: account.serverURL.host() ?? account.serverURL.absoluteString)
                            .font(.caption).foregroundStyle(Tokens.palette.mutedFg).lineLimit(1)
                    }
                    Spacer(minLength: 0)
                }
                .padding(Tokens.Spacing.md)
                .contentShape(.rect)
            }
            .buttonStyle(.plain)
            .accessibilityElement(children: .combine)
        }
        .padding(.horizontal, Tokens.Spacing.md)
        .padding(.bottom, Tokens.Spacing.md)
        .frame(width: 288)
        .frame(maxHeight: .infinity)
        .background(Tokens.palette.sidebar)
        .overlay(alignment: .trailing) { Rectangle().fill(Tokens.palette.border).frame(width: 1) }
    }
}
