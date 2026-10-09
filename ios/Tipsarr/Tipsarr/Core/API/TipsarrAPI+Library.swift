import Foundation
import OpenAPIRuntime

extension TipsarrAPI {
    func library(_ filters: LibraryFilters, page: Int, pageSize: Int = 48) async throws -> LibraryPage {
        try await load { client in
            let kind: Operations.listLibrary.Input.Query._typePayload = switch filters.kind {
            case .all: .all
            case .movie: .movie
            case .tv: .tv
            }
            let query = Operations.listLibrary.Input.Query(
                _type: kind,
                q: filters.query.isEmpty ? nil : filters.query,
                genre: filters.genres.isEmpty ? nil : filters.genres.sorted(),
                yearFrom: filters.yearFrom.map(Int64.init),
                yearTo: filters.yearTo.map(Int64.init),
                minRating: filters.minRating,
                maxRuntime: filters.maxRuntime.map(Int64.init),
                watched: .init(rawValue: filters.watched.rawValue),
                sort: .init(rawValue: filters.sort.rawValue),
                dir: filters.descending ? .desc : .asc,
                page: Int64(page),
                pageSize: Int64(pageSize)
            )
            guard case .ok(let ok) = try await client.listLibrary(query: query) else { throw APIError.unexpected }
            let body = try ok.body.json
            return LibraryPage(items: body.items.map(LibraryItem.init), page: Int(body.page), totalPages: Int(body.totalPages), total: Int(body.total))
        }
    }

    func libraryFacets() async throws -> LibraryFacets {
        try await load { client in
            guard case .ok(let ok) = try await client.libraryFacets() else { throw APIError.unexpected }
            let body = try ok.body.json
            return LibraryFacets(
                genres: body.genres.map { GenreCount(name: $0.name, count: Int($0.count)) },
                movies: Int(body.movies), shows: Int(body.shows),
                yearMin: Int(body.yearMin), yearMax: Int(body.yearMax), maxRuntimeMinutes: Int(body.maxRuntimeMinutes)
            )
        }
    }

    func watchlist() async throws -> [MediaItem] {
        try await load { client in
            guard case .ok(let ok) = try await client.list_watchlist() else { throw APIError.unexpected }
            return try ok.body.json.map(MediaItem.init)
        }
    }

    func blocklist() async throws -> [MediaItem] {
        try await load { client in
            guard case .ok(let ok) = try await client.list_blocklist() else { throw APIError.unexpected }
            return try ok.body.json.map(MediaItem.init)
        }
    }
}

extension LibraryItem {
    init(_ item: Components.Schemas.LibraryItem) {
        self.init(
            type: item._type == .tv ? .tv : .movie,
            tmdbId: Int(item.tmdbId),
            title: item.title,
            year: item.year.map(Int.init),
            rating: item.rating,
            runtimeMinutes: item.runtimeMinutes.map(Int.init),
            genres: item.genres,
            posterPath: item.posterUrl.flatMap { $0.isEmpty ? nil : $0 },
            watched: item.watched
        )
    }
}

extension TipsarrAPI {
    /// Scores for several movies at once, keyed by TMDB id.
    func movieScores(_ ids: [Int], source: RatingSource) async throws -> [Int: RatingsSummary] {
        try await load { client in
            let query = Operations.moviesRatings.Input.Query(ids: ids.map(String.init).joined(separator: ","), source: .init(rawValue: source.rawValue))
            guard case .ok(let ok) = try await client.moviesRatings(query: query) else { throw APIError.unexpected }
            var result: [Int: RatingsSummary] = [:]
            for (key, scores) in try ok.body.json.additionalProperties {
                guard let id = Int(key) else { continue }
                result[id] = RatingsSummary(tmdb: nil, imdb: scores.imdb?.value, rottenTomatoes: scores.rottenTomatoes?.value, metacritic: scores.metacritic?.value)
            }
            return result
        }
    }
}
