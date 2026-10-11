import SwiftUI

struct GenreScreen: View {
    let route: GenreRoute
    @Environment(\.appContext) private var context
    /// Movies or shows: the switch is only offered when the genre exists in both lists.
    @State private var type: MediaType

    init(route: GenreRoute) {
        self.route = route
        _type = State(initialValue: route.movieID != nil ? .movie : .tv)
    }

    private var genreID: Int? { type == .movie ? route.movieID : route.tvID }
    private var both: Bool { route.movieID != nil && route.tvID != nil }

    var body: some View {
        if let context, let genreID {
            GenreGrid(route: route, type: $type, both: both, model: MediaListModel(source: .genre(type, genreID), api: context.api))
                .id(type)
        }
    }
}

private struct GenreGrid: View {
    let route: GenreRoute
    @Binding var type: MediaType
    let both: Bool
    @State var model: MediaListModel

    var body: some View {
        ScrollView {
            if both {
                Picker("m.search.genres", selection: $type) {
                    Text("type.movies").tag(MediaType.movie)
                    Text("type.shows").tag(MediaType.tv)
                }
                .pickerStyle(.segmented)
                .padding(.horizontal, Tokens.Spacing.lg)
                .padding(.top, Tokens.Spacing.sm)
            }
            MediaGrid(model: model).padding(.vertical, Tokens.Spacing.sm)
        }
        .background(Tokens.palette.bg)
        .navigationTitle(Text(verbatim: route.name))
        .navigationBarTitleDisplayMode(.inline)
        .refreshable { await model.refresh() }
    }
}
