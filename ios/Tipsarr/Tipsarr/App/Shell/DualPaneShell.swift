import SwiftUI

/// Right pane of the unfolded iPhone Duo: the title or request picked in the list on the left.
struct DetailPaneView: View {
    let pane: DetailPane

    var body: some View {
        NavigationStack {
            switch pane.content {
            case .media(let route):
                MediaDetailScreen(route: route)
            case .request(let record):
                if let model = pane.requestsModel {
                    RequestDetailView(initial: record, model: model).id(record.id)
                } else {
                    placeholder
                }
            case nil:
                placeholder
            }
        }
        .mediaDestinations()
        .background(Tokens.palette.bg)
        // Links inside the detail push on its own stack (with a back button) instead of replacing it.
        .environment(\.detailPane, nil)
    }

    private var placeholder: some View {
        StateView(symbol: "rectangle.split.2x1", title: "m.pane.select").background(Tokens.palette.bg)
    }
}
