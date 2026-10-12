import Foundation
import OpenAPIRuntime

/// One card of the Teasers feed: a title and its trailer file.
struct Teaser: Sendable, Identifiable, Equatable {
    var item: MediaItem
    var trailerKey: String
    var watchlisted: Bool
    /// The trailer file on this server, played natively.
    var videoURL: URL

    var id: String { item.id }
}

struct TeaserFeed: Sendable {
    var items: [Teaser]
    var done: Bool
}

extension TipsarrAPI {
    /// The next unseen trailers whose file is ready. No items and `done` false: trailers are still being prepared. `exclude` holds the `type:id` keys the app already has but has not swiped past.
    func teasers(limit: Int = 8, exclude: [String]) async throws -> TeaserFeed {
        try await load { client in
            let keys = exclude.isEmpty ? nil : exclude.joined(separator: ",")
            guard case .ok(let ok) = try await client.teasers(query: .init(limit: Int64(limit), exclude: keys)) else { throw APIError.unexpected }
            let body = try ok.body.json
            return TeaserFeed(items: body.items.compactMap { Teaser($0, server: serverURL) }, done: body.done)
        }
    }

    /// The person swiped past this title: the feed never shows it again.
    func markTeaserSeen(_ type: MediaType, id: Int) async throws {
        try await load { client in
            _ = try await client.teaserSeen(body: .json(.init(tmdbId: Int64(id), _type: .init(rawValue: type.rawValue)!)))
        }
    }
}

extension Teaser {
    /// Nil when the server's video address cannot be read (the card is left out).
    init?(_ t: Components.Schemas.Teaser, server: URL) {
        guard let video = URL(string: t.videoUrl, relativeTo: server)?.absoluteURL else { return nil }
        let state: RequestState? = switch (t.availability, t.requestStatus) {
        case (.available, _): .available
        case (.partial, _): .partial
        case (_, .approved): .approved
        case (_, .pending): .requested
        default: nil
        }
        self.init(
            item: MediaItem(
                type: t._type == .tv ? .tv : .movie,
                tmdbId: Int(t.tmdbId),
                title: t.title,
                posterPath: t.posterPath,
                releaseYear: t.releaseDate.map { String($0.prefix(4)) }.flatMap { $0.isEmpty ? nil : $0 },
                voteAverage: t.voteAverage,
                state: state,
                overview: t.overview.flatMap { $0.isEmpty ? nil : $0 },
                backdropPath: t.backdropPath.flatMap { $0.isEmpty ? nil : $0 },
                genreIds: (t.genreIds ?? []).map(Int.init)
            ),
            trailerKey: t.trailerKey,
            watchlisted: t.watchlisted,
            videoURL: video
        )
    }
}

/// A title the person swiped past in Teasers, and when.
struct TeaserHistoryEntry: Sendable, Identifiable, Equatable {
    var item: MediaItem
    var seenAt: Date
    var id: String { item.id }
}

extension TipsarrAPI {
    /// The history, newest first, with how many titles it holds in all.
    func teaserHistory(offset: Int, limit: Int = 40) async throws -> (entries: [TeaserHistoryEntry], total: Int) {
        try await load { client in
            guard case .ok(let ok) = try await client.teaserHistory(query: .init(limit: Int64(limit), offset: Int64(offset))) else { throw APIError.unexpected }
            let body = try ok.body.json
            return (body.items.map(TeaserHistoryEntry.init), Int(body.total))
        }
    }

    /// The title can show up in the feed again.
    func forgetTeaser(_ type: MediaType, id: Int) async throws {
        try await load { client in
            _ = try await client.teaserForget(path: .init(_type: .init(rawValue: type.rawValue)!, id: Int64(id)))
        }
    }

    /// Wipes the whole history of the signed-in person.
    func forgetAllTeasers() async throws {
        try await load { client in _ = try await client.teaserForgetAll() }
    }
}

extension TeaserHistoryEntry {
    init(_ h: Components.Schemas.HistoryItem) {
        let state: RequestState? = switch (h.availability, h.requestStatus) {
        case (.available, _): .available
        case (.partial, _): .partial
        case (_, .approved): .approved
        case (_, .pending): .requested
        default: nil
        }
        self.init(
            item: MediaItem(
                type: h._type == .tv ? .tv : .movie,
                tmdbId: Int(h.tmdbId),
                title: h.title,
                posterPath: h.posterPath,
                releaseYear: h.releaseDate.map { String($0.prefix(4)) }.flatMap { $0.isEmpty ? nil : $0 },
                voteAverage: h.voteAverage,
                state: state,
                overview: h.overview.flatMap { $0.isEmpty ? nil : $0 },
                backdropPath: h.backdropPath.flatMap { $0.isEmpty ? nil : $0 },
                genreIds: (h.genreIds ?? []).map(Int.init)
            ),
            seenAt: Date(timeIntervalSince1970: TimeInterval(h.seenAt))
        )
    }
}
