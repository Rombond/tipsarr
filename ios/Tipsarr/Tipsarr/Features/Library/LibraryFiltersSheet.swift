import SwiftUI

struct LibraryFiltersSheet: View {
    @Binding var filters: LibraryFilters
    let facets: LibraryFacets?

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
            .filterSheetToolbar(canReset: !(filters.sheetIsDefault && filters.watched == .any)) {
                let kind = filters.kind, query = filters.query
                filters = LibraryFilters()
                filters.kind = kind
                filters.query = query
            }
        }
    }

    // MARK: Sections

    private func genres(_ facets: LibraryFacets) -> some View {
        FilterSection("library.genres") {
            GenrePickerRow(names: facets.genres.map(\.name), selection: $filters.genres, anyGenre: $filters.anyGenre)
        }
    }

    private func years(_ facets: LibraryFacets) -> some View {
        FilterYearRange(from: $filters.yearFrom, to: $filters.yearTo, range: facets.yearMin...facets.yearMax)
    }

    private var rating: some View {
        FilterMinRating(rating: $filters.minRating)
    }

    private func runtime(_ facets: LibraryFacets) -> some View {
        FilterSection("library.runtime") {
            VStack(alignment: .leading, spacing: Tokens.Spacing.xs) {
                Slider(
                    value: Binding(get: { Double(filters.maxRuntime ?? facets.maxRuntimeMinutes) },
                                   set: { filters.maxRuntime = $0 > Double(facets.maxRuntimeMinutes) - 5 ? nil : Int($0) }),
                    in: 20...Double(max(facets.maxRuntimeMinutes, 21)), step: 5
                )
                Group {
                    if let value = filters.maxRuntime { Text(verbatim: L10n.string("m.library.under", Duration.seconds(value * 60).formatted(.units(allowed: [.hours, .minutes], width: .narrow)))) } else { Text("library.any") }
                }
                .font(.footnote)
                .foregroundStyle(Tokens.palette.mutedFg)
            }
        }
    }

    private var watched: some View {
        FilterSection("library.watched_state") {
            Picker("library.watched_state", selection: $filters.watched) {
                Text("library.any").tag(WatchedFilter.any)
                Text("library.watched").tag(WatchedFilter.yes)
                Text("library.not_watched").tag(WatchedFilter.no)
            }
            .pickerStyle(.segmented)
        }
    }

    private var sort: some View {
        FilterSort(selection: $filters.sort, options: LibrarySort.allCases, title: \.title) {
            Picker("library.sort_by", selection: $filters.descending) {
                Text("library.descending").tag(true)
                Text("library.ascending").tag(false)
            }
            .pickerStyle(.segmented)
            .frame(maxWidth: 200)
        }
    }
}
