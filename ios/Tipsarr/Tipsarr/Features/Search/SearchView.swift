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
    @Environment(\.horizontalSizeClass) private var sizeClass
    @State private var model: SearchModel
    @State private var query = ""
    @State private var scope: SearchScope = .all
    @State private var filters = SearchFilters()
    @State private var showFilters = false
    @State private var width: CGFloat = 0
    /// Selected result in the two-column layout.
    @State private var selection: MediaRoute?

    init(api: TipsarrAPI) {
        _model = State(initialValue: SearchModel(api: api))
    }

    private var trimmed: String { query.trimmingCharacters(in: .whitespacesAndNewlines) }

    /// Two columns (search and results on the left, the title on the right) when the screen is wide enough.
    private var split: Bool { width >= 700 }

    var body: some View {
        Group {
            if split { splitBody } else { stackBody }
        }
        .onGeometryChange(for: CGFloat.self, of: { $0.size.width }) { width = $0 }
        .sheet(isPresented: $showFilters) {
            SearchFiltersSheet(filters: $filters, genres: model.genres, years: resultYears)
                .tipsarrSheet(detents: [.large])
                .task { await model.loadGenres() }
        }
        .task(id: query) {
            // Debounce: typing cancels this task before the request starts.
            try? await Task.sleep(for: .milliseconds(350))
            guard !Task.isCancelled else { return }
            await model.search(query)
        }
    }

    private var stackBody: some View {
        NavigationStack {
            searchable(pane, placement: .navigationBarDrawer(displayMode: .always))
                .paneInset()
                .mediaDestinations()
        }
    }

    private var splitBody: some View {
        NavigationSplitView(columnVisibility: .constant(.all)) {
            searchable(pane, placement: .sidebar)
                .navigationSplitViewColumnWidth(min: 340, ideal: 400, max: 460)
                .mediaDestinations()
        } detail: {
            NavigationStack {
                if let selection {
                    MediaDetailScreen(route: selection)
                } else {
                    StateView(symbol: "magnifyingglass", title: trimmed.isEmpty ? "m.search.start" : "m.search.select")
                        .background(Tokens.palette.bg)
                }
            }
            .mediaDestinations()
        }
        .navigationSplitViewStyle(.balanced)
        // The first result is shown as soon as the results arrive.
        .onChange(of: visibleTitles.first?.id) { selectFirstIfNeeded() }
        .onChange(of: scope) { selectFirstIfNeeded() }
    }

    private func selectFirstIfNeeded() {
        guard !trimmed.isEmpty else { return }
        if selection == nil || !visibleTitles.contains(where: { $0.route == selection }) { selection = visibleTitles.first?.route }
    }

    private var pane: some View {
        Group {
            if trimmed.isEmpty { landing } else { results }
        }
        .background(Tokens.palette.bg)
        .navigationTitle("m.tab.search")
        // Always inline: switching between large and inline titles left a stale large-title offset (a gap under the search field) when a search started from the recents.
        .navigationBarTitleDisplayMode(.inline)
        // The scopes are our own segmented control so the filter button can sit next to it.
        .safeAreaInset(edge: .top, spacing: 0) {
            if !trimmed.isEmpty { scopeBar }
        }
    }

    private var scopeBar: some View {
        HStack(spacing: Tokens.Spacing.md) {
            Picker("m.tab.search", selection: $scope) {
                ForEach(SearchScope.allCases, id: \.self) { Text($0.title).tag($0) }
            }
            .pickerStyle(.segmented)
            if showsFilters { filtersButton }
        }
        .padding(.horizontal, Tokens.Spacing.lg)
        .padding(.vertical, Tokens.Spacing.xs)
    }

    private var showsFilters: Bool { !trimmed.isEmpty && scope != .people }

    /// Same glass circle as the system "Fermer" button of the search field.
    private var filtersButton: some View {
        Button { showFilters = true } label: {
            Image(systemName: "line.3.horizontal.decrease")
                .font(.title3.weight(.medium))
                .foregroundStyle(filters.activeCount == 0 ? Tokens.palette.fg : Color.white)
                .frame(width: Tokens.Size.touchTarget, height: Tokens.Size.touchTarget)
                .contentShape(.circle)
        }
        .buttonStyle(.plain)
        .glassEffect(filters.activeCount == 0 ? .regular.interactive() : .regular.tint(.accentColor).interactive(), in: .circle)
        .accessibilityLabel(Text("m.search.filters"))
        .accessibilityValue(Text(verbatim: filters.activeCount == 0 ? "" : String(filters.activeCount)))
    }

    /// Years of the loaded titles, for the year menus of the filter sheet.
    private var resultYears: ClosedRange<Int>? {
        let years = model.items.compactMap { $0.releaseYear.flatMap { Int($0) } }
        guard let low = years.min(), let high = years.max() else { return nil }
        return low...high
    }

    private func searchable<V: View>(_ view: V, placement: SearchFieldPlacement) -> some View {
        view
            .searchable(text: $query, placement: placement, prompt: Text("m.search.prompt"))
            .onSubmit(of: .search) { model.remember(query) }
    }

    // MARK: Landing (no query)

    /// Only the recent searches are a List (swipe to delete); the genre chips stay outside it, because inside a List row one tap fires every chip link.
    private var landing: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 0) {
                if !model.recents.isEmpty { recentsList }
                genres
                    .padding(.horizontal, Tokens.Spacing.lg)
                    .padding(.top, Tokens.Spacing._2xl)
            }
            // Without the recents nothing fills the width, and the chips would sit centred instead of at the edge.
            .frame(maxWidth: .infinity, alignment: .leading)
        }
        .task { await model.loadGenres() }
    }

    private var recentsList: some View {
        let insets = EdgeInsets(top: 0, leading: Tokens.Spacing.lg, bottom: 0, trailing: Tokens.Spacing.lg)
        return List {
            HStack {
                Text("m.search.recents").font(.title3.weight(.semibold)).accessibilityAddTraits(.isHeader)
                Spacer()
                Button("m.search.clear") { model.clearRecents() }.font(.subheadline)
            }
            .frame(minHeight: Tokens.Size.touchTarget)
            .listRowInsets(insets)
            .listRowSeparator(.hidden)
            .listRowBackground(Color.clear)

            ForEach(model.recents, id: \.self) { recent in
                Button { query = recent } label: {
                    Label { Text(verbatim: recent).lineLimit(1) } icon: { Image(systemName: "clock.arrow.circlepath") }
                        .frame(maxWidth: .infinity, minHeight: Tokens.Size.touchTarget, maxHeight: Tokens.Size.touchTarget, alignment: .leading)
                        .contentShape(.rect)
                }
                .buttonStyle(.plain)
                .foregroundStyle(Tokens.palette.fg)
                .swipeActions(edge: .trailing, allowsFullSwipe: true) {
                    Button(role: .destructive) { model.removeRecent(recent) } label: { Image(systemName: "trash") }
                        .accessibilityLabel(Text("m.search.clear"))
                }
                .listRowInsets(insets)
                .listRowSeparator(.hidden)
                .listRowBackground(Color.clear)
            }
        }
        .listStyle(.plain)
        .listRowSpacing(0)
        .environment(\.defaultMinListRowHeight, 0)
        .scrollContentBackground(.hidden)
        .scrollDisabled(true)
        // The list sits in the page's scroll view, so it takes the height of its rows: the header and each row are one touch target tall.
        .frame(height: CGFloat(model.recents.count + 1) * Tokens.Size.touchTarget)
    }

    @ViewBuilder private var genres: some View {
        if !model.genres.isEmpty {
            VStack(alignment: .leading, spacing: Tokens.Spacing.lg) {
                Text("m.search.genres").font(.title3.weight(.semibold)).accessibilityAddTraits(.isHeader)
                FlowLayout(spacing: Tokens.Spacing.sm) {
                    ForEach(model.genres) { genre in
                        NavigationLink(value: genre.route) {
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
            if !hasResults && filters.activeCount > 0 && !model.items.isEmpty {
                StateView.noResults()
            } else if !hasResults {
                StateView(symbol: "magnifyingglass",
                          title: LText.verbatim(L10n.string("search.nothing", trimmed)),
                          message: "search.nothing_hint")
            } else if split {
                resultsList(titles)
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

    /// Two-column layout: results as rows, the selected one opens in the detail column.
    private func resultsList(_ titles: [MediaItem]) -> some View {
        List(selection: $selection) {
            if scope == .all || scope == .people {
                Section {
                    ForEach(scope == .people ? model.people : Array(model.people.prefix(4))) { person in
                        NavigationLink(value: person.route) { PersonRow(person: person) }
                    }
                } header: {
                    if !model.people.isEmpty { Text("type.people") }
                }
            }
            if scope != .people {
                ForEach(titles) { item in
                    SearchResultRow(item: item)
                        .tag(item.route)
                        .task { await model.loadMore(after: item) }
                }
            }
        }
        .listStyle(.plain)
        .onChange(of: selection) { if selection != nil { model.remember(query) } }
    }

    private var hasResults: Bool {
        switch scope {
        case .all: !visibleTitles.isEmpty || !model.people.isEmpty
        case .people: !model.people.isEmpty
        case .movies, .tv: !visibleTitles.isEmpty
        }
    }

    private var visibleTitles: [MediaItem] {
        let titles = switch scope {
        case .all: model.items
        case .movies: model.items.filter { $0.type == .movie }
        case .tv: model.items.filter { $0.type == .tv }
        case .people: [MediaItem]()
        }
        return filters.apply(to: titles, genreList: model.genres)
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
                MediaLink(route: item.route, requestable: item.state == nil) {
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
        navigationDestination(for: MediaRoute.self) { MediaDetailScreen(route: $0).paneInset() }
            .navigationDestination(for: PersonRoute.self) { PersonScreen(route: $0).paneInset() }
            .navigationDestination(for: GenreRoute.self) { GenreScreen(route: $0).paneInset() }
    }
}

/// One result in the two-column Search: poster, title, year, type, score, a short overview and the state.
struct SearchResultRow: View {
    let item: MediaItem

    var body: some View {
        HStack(alignment: .top, spacing: Tokens.Spacing.md) {
            RemoteImage(path: item.posterPath, size: .w185) { Image(systemName: "film").foregroundStyle(Tokens.palette.mutedFg) }
                .frame(width: 64, height: 96)
                .background(Tokens.palette.muted)
                .clipShape(.rect(cornerRadius: Tokens.Radius.sm))
                .accessibilityHidden(true)
            VStack(alignment: .leading, spacing: Tokens.Spacing.xs) {
                Text(verbatim: item.title).font(.body.weight(.semibold)).lineLimit(2)
                HStack(spacing: Tokens.Spacing.sm) {
                    Text(verbatim: [item.releaseYear, L10n.string(item.type == .tv ? "type.tv" : "type.movie")].compactMap { $0 }.joined(separator: " · "))
                        .font(.footnote).foregroundStyle(Tokens.palette.mutedFg)
                    RatingLabel(input: RatingInput(type: item.type, tmdbId: item.tmdbId, tmdb: item.voteAverage))
                }
                if let overview = item.overview {
                    Text(verbatim: overview).font(.footnote).foregroundStyle(Tokens.palette.mutedFg).lineLimit(2)
                }
                if let state = item.state { StatusBadge(state: state).padding(.top, 2) }
            }
        }
        .padding(.vertical, Tokens.Spacing.xs)
        .accessibilityElement(children: .combine)
    }
}
