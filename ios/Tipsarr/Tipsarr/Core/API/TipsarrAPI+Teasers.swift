import Foundation
import OpenAPIRuntime

/// One card of the Teasers feed: a title and the YouTube key of its trailer.
struct Teaser: Sendable, Identifiable, Equatable {
    var item: MediaItem
    var trailerKey: String
    var watchlisted: Bool
    /// A trailer file on this server plays natively; otherwise the YouTube embed plays `trailerKey`.
    var videoURL: URL?

    var id: String { item.id }
}

struct TeaserFeed: Sendable {
    var items: [Teaser]
    var done: Bool
}

extension TipsarrAPI {
    /// The next unseen trailers. `exclude` holds the `type:id` keys the app already has but has not swiped past.
    func teasers(limit: Int = 8, exclude: [String]) async throws -> TeaserFeed {
        try await load { client in
            let keys = exclude.isEmpty ? nil : exclude.joined(separator: ",")
            guard case .ok(let ok) = try await client.teasers(query: .init(limit: Int64(limit), exclude: keys)) else { throw APIError.unexpected }
            let body = try ok.body.json
            return TeaserFeed(items: body.items.map { Teaser($0, server: serverURL) }, done: body.done)
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
    init(_ t: Components.Schemas.Teaser, server: URL) {
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
            videoURL: t.player == .file ? t.videoUrl.flatMap { URL(string: $0, relativeTo: server)?.absoluteURL } : nil
        )
    }
}
