import SwiftUI

struct LibraryView: View {
    @State private var model: LibraryModel
    @State private var showFilters = false
    @AppStorage("libraryLayout") private var listLayout = false

    init(api: TipsarrAPI) {
        _model = State(initialValue: LibraryModel(api: api))
    }

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(alignment: .leading, spacing: Tokens.Spacing.lg) {
                    chips
                    content
                }
                .padding(.vertical, Tokens.Spacing.sm)
            }
            .background(Tokens.palette.bg)
            .navigationTitle("library.title")
            .searchable(text: $model.filters.query, prompt: Text("library.search"))
            .toolbar {
                ToolbarItemGroup(placement: .topBarTrailing) {
                    Button { listLayout.toggle() } label: {
                        Image(systemName: listLayout ? "square.grid.2x2" : "list.bullet")
                    }
                    .accessibilityLabel(Text(listLayout ? "m.library.grid" : "m.library.list"))
                    Button { showFilters = true } label: {
                        Image(systemName: model.filters.sheetIsDefault ? "line.3.horizontal.decrease.circle" : "line.3.horizontal.decrease.circle.fill")
                    }
                    .accessibilityLabel(Text("library.filters"))
                }
            }
            .task(id: model.filters) {
                // Typing in the search field changes `filters`: wait a moment so only the last text is sent.
                if !model.filters.query.isEmpty { try? await Task.sleep(for: .milliseconds(350)) }
                guard !Task.isCancelled else { return }
                await model.reload()
            }
            .task { await model.loadFacets() }
            .refreshable { await model.reload() }
            .mediaDestinations()
            .sheet(isPresented: $showFilters) {
                LibraryFiltersSheet(filters: $model.filters, facets: model.facets)
                    .tipsarrSheet(detents: [.large])
            }
        }
    }

    // MARK: Chips

    private var chips: some View {
        ScrollView(.horizontal, showsIndicators: false) {
            HStack(spacing: Tokens.Spacing.sm) {
                Chip(title: "library.all", isSelected: model.filters.kind == .all && model.filters.watched != .no) {
                    model.filters.kind = .all
                    model.filters.watched = .any
                }
                Chip(title: "type.movies", isSelected: model.filters.kind == .movie) { model.filters.kind = .movie }
                Chip(title: "type.shows", isSelected: model.filters.kind == .tv) { model.filters.kind = .tv }
                Chip(title: "m.library.unwatched", isSelected: model.filters.watched == .no) {
                    model.filters.watched = model.filters.watched == .no ? .any : .no
                }
            }
            .padding(.horizontal, Tokens.Spacing.lg)
        }
    }

    // MARK: Content

    @ViewBuilder private var content: some View {
        switch model.phase {
        case .idle, .loading:
            LazyVGrid(columns: Self.columns, spacing: Tokens.Spacing.lg) {
                ForEach(0..<9, id: \.self) { _ in PosterSkeleton() }
            }
            .padding(.horizontal, Tokens.Spacing.lg)
        case .failed(let error):
            ErrorState(error: error) { await model.reload() }
        case .loaded where model.neverSynced:
            StateView(symbol: "books.vertical", title: "library.title", message: "library.empty").frame(minHeight: 360)
        case .loaded where model.items.isEmpty:
            StateView.noResults().frame(minHeight: 360)
        case .loaded:
            if listLayout { list } else { grid }
            if model.loadingMore { ProgressView().frame(maxWidth: .infinity).padding() }
        }
    }

    private var grid: some View {
        LazyVGrid(columns: Self.columns, spacing: Tokens.Spacing.lg) {
            ForEach(model.items) { item in
                NavigationLink(value: item.route) {
                    PosterCard(title: item.title, subtitle: item.year.map(String.init), serverPosterPath: item.posterPath, watched: item.watched)
                }
                .buttonStyle(.plain)
                .task { await model.loadMore(after: item) }
            }
        }
        .padding(.horizontal, Tokens.Spacing.lg)
    }

    private var list: some View {
        LazyVStack(spacing: 0) {
            ForEach(model.items) { item in
                NavigationLink(value: item.route) { LibraryRow(item: item) }
                    .buttonStyle(.plain)
                    .task { await model.loadMore(after: item) }
                Divider()
            }
        }
        .padding(.horizontal, Tokens.Spacing.lg)
    }

    private static let columns = [GridItem(.adaptive(minimum: 104, maximum: 180), spacing: Tokens.Spacing.md, alignment: .top)]
}

private struct LibraryRow: View {
    let item: LibraryItem

    var body: some View {
        HStack(spacing: Tokens.Spacing.md) {
            RemoteImage(serverPath: item.posterPath) {
                Image(systemName: "film").foregroundStyle(Tokens.palette.mutedFg)
            }
            .frame(width: 56, height: 84)
            .background(Tokens.palette.muted)
            .clipShape(.rect(cornerRadius: Tokens.Radius.sm))
            .accessibilityHidden(true)
            VStack(alignment: .leading, spacing: Tokens.Spacing.xs) {
                Text(verbatim: item.title).font(.body.weight(.semibold)).lineLimit(2)
                Text(verbatim: meta).font(.footnote).foregroundStyle(Tokens.palette.mutedFg).lineLimit(1)
                if item.watched {
                    Label { Text("library.watched_state") } icon: { Image(systemName: "eye.fill") }
                        .font(.caption)
                        .foregroundStyle(Tokens.palette.mutedFg)
                }
            }
            Spacer(minLength: 0)
            if let rating = item.rating, rating > 0 {
                Text(rating, format: .number.precision(.fractionLength(1))).font(.subheadline.weight(.semibold))
            }
        }
        .padding(.vertical, Tokens.Spacing.sm)
        .frame(minHeight: Tokens.Size.touchTarget)
        .contentShape(.rect)
        .accessibilityElement(children: .combine)
    }

    private var meta: String {
        var parts: [String] = []
        if let year = item.year { parts.append(String(year)) }
        if let minutes = item.runtimeMinutes, minutes > 0 {
            parts.append(Duration.seconds(minutes * 60).formatted(.units(allowed: [.hours, .minutes], width: .narrow)))
        }
        if !item.genres.isEmpty { parts.append(item.genres.prefix(2).formatted()) }
        return parts.joined(separator: " · ")
    }
}
