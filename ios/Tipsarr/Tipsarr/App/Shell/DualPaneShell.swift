import SwiftUI

/// Right pane of the unfolded iPhone Duo: what you picked on the left, or a default for the current tab
/// (Profile: your statistics, Search: what is trending).
struct DetailPaneView: View {
    let pane: DetailPane
    let tab: AppTab
    let account: Account
    let profile: Profile
    /// Window trailing safe area (clock strip on the Duo).
    var trailingInset: CGFloat = 0

    var body: some View {
        NavigationStack {
            // Full width like the left pane: content may run behind the clock strip.
            Group { content }
                .paneInset()
                // Destinations belong inside the stack, or a tap on a poster pushes nothing.
                .mediaDestinations()
                .navigationDestination(for: ProfileRoute.self) { ProfileDestination(route: $0, account: account, profile: profile).paneInset() }
        }
        .background(Tokens.palette.bg)
        // Links inside the detail push on its own stack (with a back button) instead of replacing it.
        .environment(\.detailPane, nil)
        .environment(\.paneTrailingInset, trailingInset)
    }

    @ViewBuilder private var content: some View {
        switch pane.content {
            case .media(let route):
                MediaDetailScreen(route: route)
            case .request(let record):
                if let model = pane.requestsModel {
                    RequestDetailView(initial: record, model: model).id(record.id)
                } else {
                    placeholder
                }
            case .profile(let route):
                ProfileDestination(route: route, account: account, profile: profile)
            case nil:
                defaultContent
        }
    }

    @ViewBuilder private var defaultContent: some View {
        switch tab {
        case .profile: StatsView()
        case .search: TrendingPane()
        default: placeholder
        }
    }

    private var placeholder: some View {
        StateView(symbol: "rectangle.split.2x1", title: "m.pane.select").background(Tokens.palette.bg)
    }
}

/// Search tab, right pane, nothing picked yet: the trending titles.
struct TrendingPane: View {
    @Environment(\.appContext) private var context

    var body: some View {
        if let context {
            TrendingGrid(model: MediaListModel(source: .trending, api: context.api))
        }
    }
}

private struct TrendingGrid: View {
    @State var model: MediaListModel

    var body: some View {
        ScrollView { MediaGrid(model: model).padding(.vertical, Tokens.Spacing.sm) }
            .background(Tokens.palette.bg)
            .navigationTitle("m.pane.trending")
            .navigationBarTitleDisplayMode(.inline)
    }
}
