import SwiftUI

struct SearchFiltersSheet: View {
    @Binding var filters: SearchFilters
    let genres: [MergedGenre]
    /// Years of the loaded results: the year menus offer exactly those.
    let years: ClosedRange<Int>?
    @Environment(\.dismiss) private var dismiss

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
            .navigationTitle("m.search.filters")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItemGroup(placement: .topBarLeading) {
                    Button { dismiss() } label: { Image(systemName: "xmark") }
                        .accessibilityLabel(Text("m.library.close"))
                    Button { filters = SearchFilters() } label: { Image(systemName: "arrow.counterclockwise") }
                        .disabled(filters.activeCount == 0)
                        .accessibilityLabel(Text("library.reset"))
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button(role: .confirm) { dismiss() } label: { Image(systemName: "checkmark") }
                        .accessibilityLabel(Text("m.library.apply"))
                }
            }
        }
    }

    private func section<Content: View>(_ title: LText, @ViewBuilder content: () -> Content) -> some View {
        VStack(alignment: .leading, spacing: Tokens.Spacing.md) {
            Text(title).font(.headline).accessibilityAddTraits(.isHeader)
            content()
        }
    }

    private var genresSection: some View {
        section("library.genres") {
            GenrePickerRow(names: genres.map(\.name), selection: $filters.genres, anyGenre: $filters.anyGenre)
        }
    }

    private func yearsSection(_ range: ClosedRange<Int>) -> some View {
        section("library.year") {
            HStack(spacing: Tokens.Spacing.md) {
                yearMenu("library.year_from", selection: $filters.yearFrom, range: range)
                yearMenu("library.year_to", selection: $filters.yearTo, range: range)
            }
        }
    }

    private func yearMenu(_ title: LText, selection: Binding<Int?>, range: ClosedRange<Int>) -> some View {
        VStack(alignment: .leading, spacing: Tokens.Spacing.xs) {
            Text(title).font(.footnote.weight(.medium)).foregroundStyle(Tokens.palette.mutedFg)
            Picker(selection: selection) {
                Text("library.any").tag(Int?.none)
                ForEach(Array(range).reversed(), id: \.self) { year in Text(verbatim: String(year)).tag(Int?.some(year)) }
            } label: {
                Text(title)
            }
            .labelsHidden()
            .pickerStyle(.wheel)
            .frame(maxWidth: .infinity, alignment: .leading)
            .padding(.horizontal, Tokens.Spacing.sm)
            .frame(minHeight: Tokens.Size.touchTarget)
            .background(Tokens.palette.muted, in: .rect(cornerRadius: Tokens.Radius.md))
        }
        .frame(maxWidth: .infinity)
    }

    private var ratingSection: some View {
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

    private var availabilitySection: some View {
        section("m.search.avail") {
            Picker("m.search.avail", selection: $filters.availability) {
                Text("library.any").tag(SearchFilters.Availability.any)
                Text("m.search.avail_missing").tag(SearchFilters.Availability.missing)
                Text("m.search.avail_in").tag(SearchFilters.Availability.inLibrary)
            }
            .pickerStyle(.segmented)
        }
    }

    private var sortSection: some View {
        section("library.sort_by") {
            Picker("library.sort_by", selection: $filters.sort) {
                ForEach(SearchFilters.Sort.allCases, id: \.self) { Text($0.title).tag($0) }
            }
            .pickerStyle(.segmented)
        }
    }
}
