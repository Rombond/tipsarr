import SwiftUI

struct LibraryFiltersSheet: View {
    @Binding var filters: LibraryFilters
    let facets: LibraryFacets?
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(alignment: .leading, spacing: Tokens.Spacing._2xl) {
                    if let facets, !facets.genres.isEmpty { genres(facets) }
                    if let facets { years(facets) }
                    rating
                    if let facets, facets.maxRuntimeMinutes > 0 { runtime(facets) }
                    watched
                    sort
                }
                .padding(Tokens.Spacing.xl)
            }
            .navigationTitle("library.filters")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarLeading) {
                    Button("library.reset") {
                        let kind = filters.kind, query = filters.query
                        filters = LibraryFilters()
                        filters.kind = kind
                        filters.query = query
                    }
                    .disabled(filters.sheetIsDefault && filters.watched == .any)
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("m.library.apply") { dismiss() }
                }
            }
        }
    }

    // MARK: Sections

    private func section<Content: View>(_ title: LText, @ViewBuilder content: () -> Content) -> some View {
        VStack(alignment: .leading, spacing: Tokens.Spacing.md) {
            Text(title).font(.headline).accessibilityAddTraits(.isHeader)
            content()
        }
    }

    private func genres(_ facets: LibraryFacets) -> some View {
        section("library.genres") {
            FlowLayout(spacing: Tokens.Spacing.sm) {
                ForEach(facets.genres.prefix(30)) { genre in
                    let selected = filters.genres.contains(genre.name)
                    Button {
                        if selected { filters.genres.remove(genre.name) } else if filters.genres.count < 6 { filters.genres.insert(genre.name) }
                    } label: {
                        Text(verbatim: genre.name)
                            .font(.subheadline.weight(.medium))
                            .padding(.horizontal, Tokens.Spacing.md)
                            .frame(minHeight: Tokens.Size.touchTarget - Tokens.Spacing.sm)
                            .foregroundStyle(selected ? Tokens.palette.primaryFg : Tokens.palette.fg)
                            .background(selected ? Tokens.palette.primary : Tokens.palette.muted, in: .capsule)
                    }
                    .buttonStyle(.plain)
                    .accessibilityAddTraits(selected ? .isSelected : [])
                }
            }
            Text("library.genres_all").font(.footnote).foregroundStyle(Tokens.palette.mutedFg)
        }
    }

    private func years(_ facets: LibraryFacets) -> some View {
        section("library.year") {
            HStack(spacing: Tokens.Spacing.md) {
                yearMenu("library.year_from", selection: $filters.yearFrom, range: facets.yearMin...facets.yearMax)
                yearMenu("library.year_to", selection: $filters.yearTo, range: facets.yearMin...facets.yearMax)
            }
        }
    }

    private func yearMenu(_ title: LText, selection: Binding<Int?>, range: ClosedRange<Int>) -> some View {
        Picker(selection: selection) {
            Text("library.any").tag(Int?.none)
            ForEach(Array(range).reversed(), id: \.self) { year in Text(verbatim: String(year)).tag(Int?.some(year)) }
        } label: {
            Text(title)
        }
        .pickerStyle(.menu)
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(.horizontal, Tokens.Spacing.sm)
        .frame(minHeight: Tokens.Size.touchTarget)
        .background(Tokens.palette.muted, in: .rect(cornerRadius: Tokens.Radius.md))
    }

    private var rating: some View {
        section("library.min_rating") {
            VStack(alignment: .leading, spacing: Tokens.Spacing.xs) {
                Slider(value: Binding(get: { filters.minRating ?? 0 }, set: { filters.minRating = $0 < 0.5 ? nil : ($0 * 2).rounded() / 2 }), in: 0...9, step: 0.5)
                Group {
                    if let value = filters.minRating { Text(value, format: .number.precision(.fractionLength(1))) } else { Text("library.any") }
                }
                .font(.footnote)
                .foregroundStyle(Tokens.palette.mutedFg)
            }
        }
    }

    private func runtime(_ facets: LibraryFacets) -> some View {
        section("library.runtime") {
            VStack(alignment: .leading, spacing: Tokens.Spacing.xs) {
                Slider(
                    value: Binding(get: { Double(filters.maxRuntime ?? facets.maxRuntimeMinutes) },
                                   set: { filters.maxRuntime = Int($0) >= facets.maxRuntimeMinutes ? nil : Int($0) }),
                    in: 20...Double(max(facets.maxRuntimeMinutes, 21)), step: 5
                )
                Group {
                    if let value = filters.maxRuntime { Text(verbatim: L10n.string("library.under", String(value))) } else { Text("library.any") }
                }
                .font(.footnote)
                .foregroundStyle(Tokens.palette.mutedFg)
            }
        }
    }

    private var watched: some View {
        section("library.watched_state") {
            Picker("library.watched_state", selection: $filters.watched) {
                Text("library.any").tag(WatchedFilter.any)
                Text("library.watched").tag(WatchedFilter.yes)
                Text("library.not_watched").tag(WatchedFilter.no)
            }
            .pickerStyle(.segmented)
        }
    }

    private var sort: some View {
        section("library.sort_by") {
            HStack(spacing: Tokens.Spacing.md) {
                Picker("library.sort_by", selection: $filters.sort) {
                    ForEach(LibrarySort.allCases, id: \.self) { Text($0.title).tag($0) }
                }
                .pickerStyle(.menu)
                .frame(maxWidth: .infinity, alignment: .leading)
                Picker("library.sort_by", selection: $filters.descending) {
                    Text("library.descending").tag(true)
                    Text("library.ascending").tag(false)
                }
                .pickerStyle(.segmented)
                .frame(maxWidth: 200)
            }
        }
    }
}
