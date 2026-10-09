import SwiftUI

enum SearchScope: Hashable, CaseIterable {
    case all, movies, tv, people

    var title: LText {
        switch self {
        case .all: "search.tab_all"
        case .movies: "type.movies"
        case .tv: "type.shows"
        case .people: "type.people"
        }
    }
}

struct SearchView: View {
    @State private var model: SearchModel
    @State private var query = ""
    @State private var scope: SearchScope = .all

    init(api: TipsarrAPI) {
        _model = State(initialValue: SearchModel(api: api))
    }

    private var trimmed: String { query.trimmingCharacters(in: .whitespacesAndNewlines) }

    var body: some View {
        NavigationStack {
            Group {
                if trimmed.isEmpty { landing } else { results }
            }
            .background(Tokens.palette.bg)
            .navigationTitle("m.tab.search")
            .searchable(text: $query, prompt: Text("m.search.prompt"))
            .searchScopes($scope, activation: .onSearchPresentation) {
                ForEach(SearchScope.allCases, id: \.self) { Text($0.title).tag($0) }
            }
            .onSubmit(of: .search) { model.remember(query) }
            .task(id: query) {
                // Debounce: typing cancels this task before the request starts.
                try? await Task.sleep(for: .milliseconds(350))
                guard !Task.isCancelled else { return }
                await model.search(query)
            }
            .mediaDestinations()
        }
    }

    // MARK: Landing (no query)

    private var landing: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: Tokens.Spacing._2xl) {
                if !model.recents.isEmpty {
                    VStack(alignment: .leading, spacing: Tokens.Spacing.sm) {
                        HStack {
                            Text("m.search.recents").font(.title3.weight(.semibold)).accessibilityAddTraits(.isHeader)
                            Spacer()
                            Button("m.search.clear") { model.clearRecents() }.font(.subheadline)
                        }
                        ForEach(model.recents, id: \.self) { recent in
                            Button { query = recent } label: {
                                Label { Text(verbatim: recent) } icon: { Image(systemName: "clock.arrow.circlepath") }
                                    .frame(maxWidth: .infinity, minHeight: Tokens.Size.touchTarget, alignment: .leading)
                                    .contentShape(.rect)
                            }
                            .buttonStyle(.plain)
                            .foregroundStyle(Tokens.palette.fg)
                        }
                    }
                }
                genres
            }
            .padding(Tokens.Spacing.lg)
        }
        .task { await model.loadGenres() }
    }

    @ViewBuilder private var genres: some View {
        if !model.movieGenres.isEmpty || !model.tvGenres.isEmpty {
            VStack(alignment: .leading, spacing: Tokens.Spacing.lg) {
                Text("m.search.genres").font(.title3.weight(.semibold)).accessibilityAddTraits(.isHeader)
                genreGroup("type.movies", genres: model.movieGenres, type: .movie)
                genreGroup("type.shows", genres: model.tvGenres, type: .tv)
            }
        }
    }

    private func genreGroup(_ title: LText, genres: [Genre], type: MediaType) -> some View {
        VStack(alignment: .leading, spacing: Tokens.Spacing.sm) {
            Text(title).font(.subheadline.weight(.medium)).foregroundStyle(Tokens.palette.mutedFg)
            FlowLayout(spacing: Tokens.Spacing.sm) {
                ForEach(genres) { genre in
                    NavigationLink(value: GenreRoute(type: type, id: genre.id, name: genre.name)) {
                        Text(verbatim: genre.name)
                            .font(.subheadline.weight(.medium))
                            .padding(.horizontal, Tokens.Spacing.md)
                            .frame(minHeight: Tokens.Size.touchTarget - Tokens.Spacing.sm)
                            .background(Tokens.palette.muted, in: .capsule)
                            .foregroundStyle(Tokens.palette.fg)
                    }
                }
            }
        }
    }

    // MARK: Results

    @ViewBuilder private var results: some View {
        switch model.phase {
        case .idle, .loading:
            ScrollView {
                LazyVGrid(columns: Self.columns, spacing: Tokens.Spacing.lg) {
                    ForEach(0..<9, id: \.self) { _ in PosterSkeleton() }
                }
                .padding(Tokens.Spacing.lg)
            }
        case .failed(let error):
            ErrorState(error: error) { await model.search(query) }
        case .loaded:
            let titles = visibleTitles
            if !hasResults {
                StateView(symbol: "magnifyingglass",
                          title: LText.verbatim(L10n.string("search.nothing", trimmed)),
                          message: "search.nothing_hint")
            } else {
                ScrollView {
                    VStack(alignment: .leading, spacing: Tokens.Spacing.xl) {
                        if scope == .all || scope == .people { peopleSection }
                        if scope != .people { titlesGrid(titles) }
                    }
                    .padding(.vertical, Tokens.Spacing.sm)
                }
            }
        }
    }

    private var hasResults: Bool {
        switch scope {
        case .all: !model.items.isEmpty || !model.people.isEmpty
        case .people: !model.people.isEmpty
        case .movies, .tv: !visibleTitles.isEmpty
        }
    }

    private var visibleTitles: [MediaItem] {
        switch scope {
        case .all: model.items
        case .movies: model.items.filter { $0.type == .movie }
        case .tv: model.items.filter { $0.type == .tv }
        case .people: []
        }
    }

    @ViewBuilder private var peopleSection: some View {
        if !model.people.isEmpty {
            VStack(alignment: .leading, spacing: Tokens.Spacing.md) {
                Text("type.people").font(.title3.weight(.semibold)).padding(.horizontal, Tokens.Spacing.lg)
                    .accessibilityAddTraits(.isHeader)
                if scope == .people {
                    VStack(spacing: 0) {
                        ForEach(model.people) { person in
                            NavigationLink(value: person.route) { PersonRow(person: person) }.buttonStyle(.plain)
                        }
                    }
                    .padding(.horizontal, Tokens.Spacing.lg)
                } else {
                    ScrollView(.horizontal, showsIndicators: false) {
                        LazyHStack(alignment: .top, spacing: Tokens.Spacing.md) {
                            ForEach(model.people.prefix(12)) { person in
                                NavigationLink(value: person.route) { PersonBubble(person: person) }.buttonStyle(.plain)
                            }
                        }
                        .padding(.horizontal, Tokens.Spacing.lg)
                    }
                }
            }
        }
    }

    private func titlesGrid(_ titles: [MediaItem]) -> some View {
        LazyVGrid(columns: Self.columns, spacing: Tokens.Spacing.lg) {
            ForEach(titles) { item in
                NavigationLink(value: item.route) {
                    PosterCard(item: item)
                }
                .buttonStyle(.plain)
                .simultaneousGesture(TapGesture().onEnded { model.remember(query) })
                .task { await model.loadMore(after: item) }
            }
        }
        .padding(.horizontal, Tokens.Spacing.lg)
    }

    private static let columns = [GridItem(.adaptive(minimum: 104, maximum: 180), spacing: Tokens.Spacing.md, alignment: .top)]
}

struct PersonBubble: View {
    let person: PersonSummary

    var body: some View {
        VStack(spacing: Tokens.Spacing.xs) {
            PersonPhoto(path: person.profilePath).frame(width: 72, height: 72)
            Text(verbatim: person.name).font(.caption.weight(.medium)).lineLimit(1)
            if let department = person.department, !department.isEmpty {
                Text(verbatim: department).font(.caption2).foregroundStyle(Tokens.palette.mutedFg).lineLimit(1)
            }
        }
        .frame(width: 88)
        .accessibilityElement(children: .combine)
    }
}

struct PersonRow: View {
    let person: PersonSummary

    var body: some View {
        HStack(spacing: Tokens.Spacing.md) {
            PersonPhoto(path: person.profilePath).frame(width: 52, height: 52)
            VStack(alignment: .leading, spacing: 2) {
                Text(verbatim: person.name).font(.body.weight(.medium))
                if let department = person.department, !department.isEmpty {
                    Text(verbatim: department).font(.footnote).foregroundStyle(Tokens.palette.mutedFg)
                }
            }
            Spacer()
            Image(systemName: "chevron.right").font(.footnote).foregroundStyle(Tokens.palette.mutedFg)
        }
        .frame(minHeight: Tokens.Size.touchTarget + Tokens.Spacing.md)
        .contentShape(.rect)
        .accessibilityElement(children: .combine)
    }
}

struct PersonPhoto: View {
    let path: String?

    var body: some View {
        RemoteImage(path: path, size: .w185) {
            Image(systemName: "person.fill").foregroundStyle(Tokens.palette.mutedFg)
        }
        .background(Tokens.palette.muted)
        .clipShape(.circle)
        .accessibilityHidden(true)
    }
}

extension View {
    /// Navigation values every stack that shows titles must handle.
    func mediaDestinations() -> some View {
        navigationDestination(for: MediaRoute.self) { MediaDetailScreen(route: $0) }
            .navigationDestination(for: PersonRoute.self) { PersonScreen(route: $0) }
            .navigationDestination(for: GenreRoute.self) { GenreScreen(route: $0) }
    }
}
