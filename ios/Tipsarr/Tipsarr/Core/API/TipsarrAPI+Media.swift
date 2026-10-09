import Foundation
import OpenAPIRuntime

extension TipsarrAPI {
    func detail(_ type: MediaType, id: Int) async throws -> MediaDetail {
        try await load { client in
            guard case .ok(let ok) = try await client.mediaDetail(path: .init(_type: .init(rawValue: type.rawValue)!, id: Int64(id))) else { throw APIError.unexpected }
            return MediaDetail(try ok.body.json)
        }
    }

    func ratings(_ type: MediaType, id: Int) async throws -> RatingsSummary {
        try await load { client in
            let scores: Components.Schemas.MovieScores
            switch type {
            case .movie:
                guard case .ok(let ok) = try await client.movieRatings(path: .init(id: Int64(id))) else { throw APIError.unexpected }
                scores = try ok.body.json
            case .tv:
                guard case .ok(let ok) = try await client.showRatings(path: .init(id: Int64(id))) else { throw APIError.unexpected }
                scores = try ok.body.json
            }
            return RatingsSummary(tmdb: nil, imdb: scores.imdb?.value, rottenTomatoes: scores.rottenTomatoes?.value,
                                  metacritic: scores.metacritic?.value)
        }
    }

    func flags(_ type: MediaType, id: Int) async throws -> TitleFlags {
        try await load { client in
            guard case .ok(let ok) = try await client.mediaFlags(path: .init(_type: .init(rawValue: type.rawValue)!, id: Int64(id))) else { throw APIError.unexpected }
            let body = try ok.body.json
            return TitleFlags(watchlisted: body.watchlisted ?? false, blocklisted: body.blocklisted ?? false)
        }
    }

    /// Nil when the server has nothing to choose from (no Radarr/Sonarr configured for the user).
    func requestOptions(_ type: MediaType) async throws -> RequestOptions? {
        do {
            return try await load { client in
                guard case .ok(let ok) = try await client.requestOptions(query: .init(_type: .init(rawValue: type.rawValue)!)) else { throw APIError.unexpected }
                let body = try ok.body.json
                return RequestOptions(
                    instanceName: body.instanceName,
                    profiles: body.profiles.map { QualityProfile(id: Int($0.id), name: $0.name) },
                    defaultProfileID: Int(body.qualityProfileId),
                    rootFolders: body.rootFolders.map { RootFolderOption(id: Int($0.id), path: $0.path, freeSpace: $0.freeSpace) },
                    defaultFolder: body.rootFolder
                )
            }
        } catch let error as APIError where error.status == 403 || error.status == 404 || error.code == "no_instance" {
            return nil
        }
    }

    func createRequest(_ type: MediaType, tmdbId: Int, seasons: [Int]?, profileID: Int?, folder: String?) async throws -> RequestRecord {
        try await load { client in
            let body = Components.Schemas.CreateRequestRequest(
                qualityProfileId: profileID.map(Int64.init),
                rootFolder: folder,
                seasons: seasons?.map(Int64.init),
                tmdbId: Int64(tmdbId),
                _type: type == .tv ? .tv : .movie
            )
            guard case .created(let ok) = try await client.createRequest(body: .json(body)) else { throw APIError.unexpected }
            return RequestRecord(try ok.body.json)
        }
    }

    func myRequests() async throws -> [RequestRecord] {
        try await load { client in
            guard case .ok(let ok) = try await client.listRequests(query: .init(filter: .mine, take: 100)) else { throw APIError.unexpected }
            let list = try ok.body.json
            return list.items.map(RequestRecord.init)
        }
    }

    func deleteRequest(id: String) async throws {
        try await load { client in
            _ = try await client.deleteRequest(path: .init(id: id))
        }
    }

    func reportIssue(_ type: MediaType, tmdbId: Int, kind: IssueKind, message: String) async throws {
        try await load { client in
            let body = Components.Schemas.CreateIssueRequest(
                kind: Components.Schemas.CreateIssueRequest.kindPayload(rawValue: kind.rawValue) ?? .other,
                message: message,
                tmdbId: Int64(tmdbId),
                _type: type == .tv ? .tv : .movie
            )
            _ = try await client.createIssue(body: .json(body))
        }
    }
}

extension MediaDetail {
    init(_ d: Components.Schemas.Detail) {
        let availability: Availability = switch d.availability {
        case .available: .available
        case .partial: .partial
        default: .none
        }
        self.init(
            type: d._type == .tv ? .tv : .movie,
            tmdbId: Int(d.tmdbId),
            title: d.title,
            tagline: d.tagline.flatMap { $0.isEmpty ? nil : $0 },
            overview: d.overview.flatMap { $0.isEmpty ? nil : $0 },
            posterPath: d.posterPath,
            backdropPath: d.backdropPath,
            releaseDate: d.releaseDate,
            runtimeMinutes: d.runtimeMinutes.map(Int.init),
            voteAverage: d.voteAverage,
            voteCount: Int(d.voteCount ?? 0),
            genres: d.genres.map(\.name),
            status: d.status,
            originalLanguage: d.originalLanguage,
            availability: availability,
            requestOpen: d.requestStatus != nil,
            seasons: (d.seasons ?? []).map {
                SeasonInfo(number: Int($0.number), name: $0.name, episodeCount: Int($0.episodeCount), airDate: $0.airDate)
            },
            numberOfSeasons: d.numberOfSeasons.map(Int.init),
            numberOfEpisodes: d.numberOfEpisodes.map(Int.init),
            studios: d.studios ?? [],
            cast: d.cast.map { CastMember(id: Int($0.id), name: $0.name, character: $0.character, profilePath: $0.profilePath) },
            recommendations: d.recommendations.map(MediaItem.init),
            watchURL: d.watchUrl.flatMap(URL.init(string:)),
            trailerKey: d.trailerKey
        )
    }
}

extension RequestRecord {
    init(_ v: Components.Schemas.View) {
        self.init(
            id: v.id,
            type: v._type == .tv ? .tv : .movie,
            tmdbId: Int(v.tmdbId ?? 0),
            title: v.title ?? "",
            posterPath: v.posterPath,
            state: Self.state(stage: v.stage.rawValue, status: v.status.rawValue),
            seasons: (v.seasons ?? []).map(Int.init),
            progressPercent: v.progress.map { Int($0.percent) },
            etaSeconds: v.progress.map { Int($0.etaSeconds) },
            declineReason: v.declineReason.flatMap { $0.isEmpty ? nil : $0 },
            error: v.error.flatMap { $0.isEmpty ? nil : $0 },
            requestedBy: v.requestedBy.name,
            decidedBy: v.decidedBy?.name,
            dryRun: v.dryRun ?? false,
            createdAt: Date(timeIntervalSince1970: TimeInterval(v.createdAt ?? 0))
        )
    }

    /// `stage` is the fine-grained state; older servers only send `status`.
    static func state(stage: String?, status: String) -> RequestState {
        if let stage, let state = RequestState(rawValue: stage) { return state }
        switch status {
        case "approved": return .approved
        case "declined": return .declined
        case "failed": return .failed
        case "available": return .available
        default: return .requested
        }
    }
}

extension TipsarrAPI {
    func setWatchlisted(_ on: Bool, _ type: MediaType, id: Int) async throws {
        try await load { client in
            if on {
                _ = try await client.add_watchlist(body: .json(.init(tmdbId: Int64(id), _type: .init(rawValue: type.rawValue)!)))
            } else {
                _ = try await client.remove_watchlist(path: .init(_type: .init(rawValue: type.rawValue)!, id: Int64(id)))
            }
        }
    }

    func setBlocklisted(_ on: Bool, _ type: MediaType, id: Int) async throws {
        try await load { client in
            if on {
                _ = try await client.add_blocklist(body: .json(.init(tmdbId: Int64(id), _type: .init(rawValue: type.rawValue)!)))
            } else {
                _ = try await client.remove_blocklist(path: .init(_type: .init(rawValue: type.rawValue)!, id: Int64(id)))
            }
        }
    }

    func retryRequest(id: String) async throws -> RequestRecord {
        try await load { client in
            guard case .ok(let ok) = try await client.retryRequest(path: .init(id: id)) else { throw APIError.unexpected }
            return RequestRecord(try ok.body.json)
        }
    }
}
