import Foundation
import OpenAPIRuntime

extension TipsarrAPI {
    func trending(page: Int) async throws -> MediaPage {
        try await load { client in
            guard case .ok(let ok) = try await client.discoverTrending(query: .init(page: Int64(page))) else { throw APIError.unexpected }
            return MediaPage(try ok.body.json)
        }
    }

    func upcoming(page: Int) async throws -> MediaPage {
        try await load { client in
            guard case .ok(let ok) = try await client.discoverUpcoming(query: .init(page: Int64(page))) else { throw APIError.unexpected }
            return MediaPage(try ok.body.json)
        }
    }

    func popularMovies(page: Int) async throws -> MediaPage {
        try await load { client in
            guard case .ok(let ok) = try await client.discover_movie(query: .init(page: Int64(page))) else { throw APIError.unexpected }
            return MediaPage(try ok.body.json)
        }
    }

    func popularTV(page: Int) async throws -> MediaPage {
        try await load { client in
            guard case .ok(let ok) = try await client.discover_tv(query: .init(page: Int64(page))) else { throw APIError.unexpected }
            return MediaPage(try ok.body.json)
        }
    }

    func suggestions() async throws -> [SuggestionRow] {
        try await load { client in
            guard case .ok(let ok) = try await client.suggestions() else { throw APIError.unexpected }
            return try ok.body.json.rows.map { row in
                SuggestionRow(id: row.id, title: row.title, items: row.items.map(MediaItem.init))
            }
        }
    }
}

extension MediaPage {
    init(_ list: Components.Schemas.List) {
        self.init(items: list.items.map(MediaItem.init), page: Int(list.page), totalPages: Int(list.totalPages))
    }
}

extension MediaItem {
    init(_ item: Components.Schemas.Item) {
        let state: RequestState? = switch (item.availability, item.requestStatus) {
        case (.available, _): .available
        case (.partial, _): .partial
        case (_, .approved): .approved
        case (_, .pending): .requested
        default: nil
        }
        self.init(
            type: item._type == .tv ? .tv : .movie,
            tmdbId: Int(item.tmdbId),
            title: item.title,
            posterPath: item.posterPath,
            releaseYear: item.releaseDate.map { String($0.prefix(4)) }.flatMap { $0.isEmpty ? nil : $0 },
            voteAverage: item.voteAverage,
            state: state
        )
    }
}
