import SwiftUI

extension AppTab {
    var title: LText {
        switch self {
        case .discover: "m.tab.discover"
        case .search: "m.tab.search"
        case .requests: "m.tab.requests"
        case .library: "m.tab.library"
        case .profile: "m.tab.profile"
        }
    }

    var symbol: String {
        switch self {
        case .discover: "safari"
        case .search: "magnifyingglass"
        case .requests: "checklist"
        case .library: "books.vertical"
        case .profile: "person.crop.circle"
        }
    }

    /// Same order as the iPhone bar, where Search is the separate last tab.
    static let ordered: [AppTab] = [.discover, .requests, .library, .profile, .search]
}

/// The tab bar of a portrait iPad window: a floating pill at the bottom, as in the Penpot iPad page.
struct FloatingTabBar: View {
    @Binding var selection: AppTab
    var badges: [AppTab: Int] = [:]
    /// Icons only when the bar is narrow (the left pane of the Duo).
    var showsTitles = true

    var body: some View {
        HStack(spacing: 0) {
            ForEach(AppTab.ordered, id: \.self) { tab in
                Button { selection = tab } label: {
                    VStack(spacing: 2) {
                        Image(systemName: tab.symbol).font(.title3.weight(selection == tab ? .semibold : .regular))
                        if showsTitles { Text(tab.title).font(.caption2.weight(selection == tab ? .semibold : .medium)) }
                    }
                    .foregroundStyle(selection == tab ? Tokens.palette.fg : Tokens.palette.mutedFg)
                    .frame(maxWidth: .infinity, minHeight: 54)
                    .background { if selection == tab { Capsule().fill(Tokens.palette.muted) } }
                    .overlay(alignment: .topTrailing) { CountBadge(count: badges[tab] ?? 0).offset(x: -10, y: 2) }
                    .contentShape(.capsule)
                }
                .buttonStyle(.plain)
                .accessibilityLabel(Text(tab.title))
                .accessibilityAddTraits(selection == tab ? .isSelected : [])
            }
        }
        .padding(Tokens.Spacing.sm)
        .frame(maxWidth: 600)
        .background(Tokens.palette.card, in: .capsule)
        .overlay { Capsule().strokeBorder(Tokens.palette.border) }
        .shadow(color: .black.opacity(0.12), radius: 16, y: 6)
        .padding(.horizontal, showsTitles ? Tokens.Spacing._3xl : Tokens.Spacing.lg)
    }
}

/// Small red count shown on a tab (pending requests for admins).
struct CountBadge: View {
    let count: Int

    var body: some View {
        if count > 0 {
            Text(verbatim: count > 99 ? "99+" : String(count))
                .font(.caption2.weight(.bold))
                .foregroundStyle(.white)
                .padding(.horizontal, 6)
                .frame(minWidth: 18, minHeight: 18)
                .background(Tokens.palette.destructive, in: .capsule)
                .accessibilityHidden(true)
        }
    }
}
