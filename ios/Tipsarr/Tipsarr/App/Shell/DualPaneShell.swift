import SwiftUI

/// Right pane of the unfolded iPhone Duo: what you picked on the left, or a default for the current tab
/// (Profile: your statistics, Search: what is trending).
struct DetailPaneView: View {
    let pane: DetailPane
    let tab: AppTab
    let account: Account
    let profile: Profile

    var body: some View {
        NavigationStack {
            // Cut at the clock strip: carousels do not run behind it.
            Group { content }
                .mask { Rectangle().ignoresSafeArea(.container, edges: [.top, .bottom, .leading]) }
        }
        .mediaDestinations()
        .navigationDestination(for: ProfileRoute.self) { ProfileDestination(route: $0, account: account, profile: profile) }
        .background(Tokens.palette.bg)
        // Links inside the detail push on its own stack (with a back button) instead of replacing it.
        .environment(\.detailPane, nil)
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
