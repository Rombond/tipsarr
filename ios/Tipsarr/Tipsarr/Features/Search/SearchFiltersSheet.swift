import SwiftUI

struct SearchFiltersSheet: View {
    @Binding var filters: SearchFilters
    let genres: [MergedGenre]
    /// Years of the loaded results: the year menus offer exactly those.
    let years: ClosedRange<Int>?

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(alignment: .leading, spacing: Tokens.Spacing._2xl) {
                    if !genres.isEmpty { genresSection }
                    if let years { yearsSection(years) }
                    ratingSection
                    availabilitySection
                    sortSection
                }
                .padding(Tokens.Spacing.xl)
            }
            .filterSheetToolbar(canReset: filters.activeCount > 0) { filters = SearchFilters() }
        }
    }

    private var genresSection: some View {
        FilterSection("library.genres") {
            GenrePickerRow(names: genres.map(\.name), selection: $filters.genres, anyGenre: $filters.anyGenre)
        }
    }

    private func yearsSection(_ range: ClosedRange<Int>) -> some View {
        FilterYearRange(from: $filters.yearFrom, to: $filters.yearTo, range: range)
    }

    private var ratingSection: some View {
        FilterMinRating(rating: $filters.minRating)
    }

    private var availabilitySection: some View {
        FilterSection("m.search.avail") {
            Picker("m.search.avail", selection: $filters.availability) {
                Text("library.any").tag(SearchFilters.Availability.any)
                Text("m.search.avail_missing").tag(SearchFilters.Availability.missing)
                Text("m.search.avail_in").tag(SearchFilters.Availability.inLibrary)
            }
            .pickerStyle(.segmented)
        }
    }

    private var sortSection: some View {
        FilterSort(selection: $filters.sort, options: Array(SearchFilters.Sort.allCases), title: \.title) { EmptyView() }
    }
}
