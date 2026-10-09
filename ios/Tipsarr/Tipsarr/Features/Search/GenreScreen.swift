import SwiftUI

struct GenreScreen: View {
    let route: GenreRoute
    @Environment(\.appContext) private var context

    var body: some View {
        if let context {
            GenreGrid(route: route, model: MediaListModel(source: .genre(route.type, route.id), api: context.api))
        }
    }
}

private struct GenreGrid: View {
    let route: GenreRoute
    @State var model: MediaListModel

    var body: some View {
        ScrollView { MediaGrid(model: model).padding(.vertical, Tokens.Spacing.sm) }
            .background(Tokens.palette.bg)
            .navigationTitle(Text(verbatim: route.name))
            .navigationBarTitleDisplayMode(.inline)
            .refreshable { await model.refresh() }
    }
}
