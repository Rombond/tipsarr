import Foundation
import SwiftUI

/// A title that is in the Jellyfin library. Posters come from Jellyfin through the server.
struct LibraryItem: Sendable, Hashable, Identifiable {
    var type: MediaType
    var tmdbId: Int
    var title: String
    var year: Int?
    var rating: Double?
    var runtimeMinutes: Int?
    var genres: [String]
    /// Server-relative path (`/api/v1/images/jellyfin/<id>?tag=<tag>`).
    var posterPath: String?
    var watched: Bool

    var id: String { "\(type.rawValue)-\(tmdbId)" }
    var route: MediaRoute { MediaRoute(type: type, tmdbId: tmdbId, title: title) }
}

struct LibraryPage: Sendable {
    var items: [LibraryItem]
    var page: Int
    var totalPages: Int
    var total: Int
    var hasMore: Bool { page < totalPages }
}

struct GenreCount: Sendable, Hashable, Identifiable {
    var name: String
    var count: Int
    var id: String { name }
}

struct LibraryFacets: Sendable {
    var genres: [GenreCount]
    var movies: Int
    var shows: Int
    var yearMin: Int
    var yearMax: Int
    var maxRuntimeMinutes: Int
}

enum LibrarySort: String, CaseIterable, Sendable {
    case added, title, year, rating, runtime, popular

    var title: LText {
        switch self {
        case .added: "library.sort_added"
        case .title: "library.sort_title"
        case .year: "library.sort_year"
        case .rating: "library.sort_rating"
        case .runtime: "library.sort_runtime"
        case .popular: "library.sort_popular"
        }
    }
}

enum WatchedFilter: String, CaseIterable, Sendable {
    case any, yes, no
}

/// Everything the filters sheet and the chips can change.
struct LibraryFilters: Sendable, Equatable {
    enum Kind: Sendable { case all, movie, tv }
    var kind: Kind = .all
    var query = ""
    var genres: Set<String> = []
    var yearFrom: Int?
    var yearTo: Int?
    var minRating: Double?
    var maxRuntime: Int?
    var watched: WatchedFilter = .any
    var sort: LibrarySort = .added
    var descending = true

    /// Filters from the sheet only (not the chips or the search text).
    var sheetIsDefault: Bool {
        genres.isEmpty && yearFrom == nil && yearTo == nil && minRating == nil && maxRuntime == nil && sort == .added && descending
    }

    var activeSheetCount: Int {
        [!genres.isEmpty, yearFrom != nil || yearTo != nil, minRating != nil, maxRuntime != nil, sort != .added || !descending].filter { $0 }.count
    }
}
