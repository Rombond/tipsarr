import Foundation

/// Filters of the search results. They work on the results already loaded, nothing is sent to the server.
struct SearchFilters: Sendable, Equatable {
    enum Availability: Sendable { case any, missing, inLibrary }
    enum Sort: Sendable, CaseIterable {
        case relevance, rating, year

        var title: LText {
            switch self {
            case .relevance: "m.search.sort_relevance"
            case .rating: "library.sort_rating"
            case .year: "library.sort_year"
            }
        }
    }

    var yearFrom: Int?
    var yearTo: Int?
    var minRating: Double?
    var availability: Availability = .any
    var genres: Set<String> = []
    /// A title needs one of the genres instead of all of them.
    var anyGenre = false
    var sort: Sort = .relevance

    var activeCount: Int {
        [yearFrom != nil || yearTo != nil, minRating != nil, availability != .any, !genres.isEmpty, sort != .relevance].filter { $0 }.count
    }

    func apply(to items: [MediaItem], genreList: [MergedGenre]) -> [MediaItem] {
        guard activeCount > 0 else { return items }
        let wanted = genreList.filter { genres.contains($0.name) }
        var out = items.filter { item in
            if yearFrom != nil || yearTo != nil {
                guard let year = item.releaseYear.flatMap({ Int($0) }) else { return false }
                if let yearFrom, year < yearFrom { return false }
                if let yearTo, year > yearTo { return false }
            }
            if let minRating, item.voteAverage < minRating { return false }
            let inLibrary = item.state == .available || item.state == .partial
            switch availability {
            case .any: break
            case .missing: if inLibrary { return false }
            case .inLibrary: if !inLibrary { return false }
            }
            if !genres.isEmpty {
                let matches = wanted.map { genre in genre.id(for: item.type).map(item.genreIds.contains) ?? false }
                if anyGenre ? !matches.contains(true) : (matches.count < genres.count || matches.contains(false)) { return false }
            }
            return true
        }
        switch sort {
        case .relevance: break
        case .rating: out.sort { $0.voteAverage > $1.voteAverage }
        case .year: out.sort { (Int($0.releaseYear ?? "") ?? 0) > (Int($1.releaseYear ?? "") ?? 0) }
        }
        return out
    }
}
