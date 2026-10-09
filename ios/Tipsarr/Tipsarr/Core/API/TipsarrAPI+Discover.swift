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

    func suggestions() async throws -> SuggestionResult {
        try await load { client in
            guard case .ok(let ok) = try await client.suggestions() else { throw APIError.unexpected }
            let body = try ok.body.json
            return SuggestionResult(
                rows: body.rows.map { SuggestionRow(id: $0.id, title: $0.title, items: $0.items.map(MediaItem.init)) },
                generating: body.generating
            )
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
            state: state,
            overview: item.overview.flatMap { $0.isEmpty ? nil : $0 },
            backdropPath: item.backdropPath.flatMap { $0.isEmpty ? nil : $0 }
        )
    }
}

extension TipsarrAPI {
    func genres(_ type: MediaType) async throws -> [Genre] {
        try await load { client in
            guard case .ok(let ok) = try await client.discoverGenres(path: .init(_type: .init(rawValue: type.rawValue)!)) else { throw APIError.unexpected }
            return try ok.body.json.map { Genre(id: Int($0.id), name: $0.name) }
        }
    }

    func byGenre(_ type: MediaType, genre: Int, page: Int) async throws -> MediaPage {
        try await load { client in
            switch type {
            case .movie:
                guard case .ok(let ok) = try await client.discover_movie(query: .init(page: Int64(page), genre: Int64(genre))) else { throw APIError.unexpected }
                return MediaPage(try ok.body.json)
            case .tv:
                guard case .ok(let ok) = try await client.discover_tv(query: .init(page: Int64(page), genre: Int64(genre))) else { throw APIError.unexpected }
                return MediaPage(try ok.body.json)
            }
        }
    }

    func search(_ query: String, page: Int) async throws -> SearchPage {
        try await load { client in
            guard case .ok(let ok) = try await client.search(query: .init(page: Int64(page), q: query)) else { throw APIError.unexpected }
            let body = try ok.body.json
            return SearchPage(
                items: body.items.map(MediaItem.init),
                people: body.people.map { PersonSummary(id: Int($0.id), name: $0.name, department: $0.department, profilePath: $0.profilePath) },
                page: Int(body.page),
                totalPages: Int(body.totalPages)
            )
        }
    }

    func person(id: Int) async throws -> PersonDetail {
        try await load { client in
            guard case .ok(let ok) = try await client.person(path: .init(id: Int64(id))) else { throw APIError.unexpected }
            let body = try ok.body.json
            return PersonDetail(
                id: Int(body.id), name: body.name, department: body.department,
                biography: body.biography.flatMap { $0.isEmpty ? nil : $0 },
                birthday: body.birthday, birthplace: body.birthplace, deathday: body.deathday,
                profilePath: body.profilePath, credits: (body.credits ?? []).map(MediaItem.init)
            )
        }
    }
}
