import Foundation
import SwiftUI

enum Availability: Sendable { case none, partial, available }

struct SeasonInfo: Sendable, Hashable, Identifiable {
    var number: Int
    var name: String
    var episodeCount: Int
    var airDate: String?
    var id: Int { number }
}

struct EpisodeInfo: Sendable, Hashable, Identifiable {
    var number: Int
    var name: String
    var overview: String?
    var airDate: String?
    var runtimeMinutes: Int?
    var stillPath: String?
    var voteAverage: Double
    var id: Int { number }
}

struct CastMember: Sendable, Hashable, Identifiable {
    var id: Int
    var name: String
    var character: String?
    var profilePath: String?
}

struct MediaDetail: Sendable {
    var type: MediaType
    var tmdbId: Int
    var title: String
    var tagline: String?
    var overview: String?
    var posterPath: String?
    var backdropPath: String?
    var releaseDate: String?
    var runtimeMinutes: Int?
    var voteAverage: Double
    var voteCount: Int
    var genres: [String]
    var status: String?
    var originalLanguage: String?
    var availability: Availability
    /// `pending` or `approved` while the signed-in user's (or anyone's) request is open.
    var requestOpen: Bool
    var seasons: [SeasonInfo]
    var numberOfSeasons: Int?
    var numberOfEpisodes: Int?
    var studios: [String]
    var cast: [CastMember]
    var recommendations: [MediaItem]
    var watchURL: URL?
    var trailerKey: String?

    var year: String? { releaseDate.flatMap { $0.count >= 4 ? String($0.prefix(4)) : nil } }
    var route: MediaRoute { MediaRoute(type: type, tmdbId: tmdbId, title: title) }
}

/// Scores out of their own scale: TMDB 0-10, IMDb 0-10, RT and Metacritic 0-100.
struct RatingsSummary: Sendable, Equatable {
    var tmdb: Double?
    var imdb: Double?
    var rottenTomatoes: Double?
    var metacritic: Double?
}

struct QualityProfile: Sendable, Hashable, Identifiable {
    var id: Int
    var name: String
}

struct RootFolderOption: Sendable, Hashable, Identifiable {
    var id: Int
    var path: String
    var freeSpace: Int64
}

struct RequestOptions: Sendable {
    var instanceName: String
    var profiles: [QualityProfile]
    var defaultProfileID: Int
    var rootFolders: [RootFolderOption]
    var defaultFolder: String
}

struct SeasonProgress: Sendable, Hashable {
    var season: Int
    var state: RequestState
}

/// A request as the apps show it.
struct RequestRecord: Sendable, Identifiable, Hashable {
    var id: String
    var type: MediaType
    var tmdbId: Int
    var title: String
    var posterPath: String?
    var state: RequestState
    var seasons: [Int]
    var progressPercent: Int?
    var etaSeconds: Int?
    /// Shows only: completion per season (season number to percent).
    var seasonProgress: [Int: Int] = [:]
    var declineReason: String?
    var error: String?
    var requestedBy: String?
    var decidedBy: String?
    var dryRun: Bool
    var createdAt: Date

    /// The owner may still cancel until the title is available.
    var isOpen: Bool { state != .available && state != .declined && state != .failed }
}

struct TitleFlags: Sendable, Equatable {
    var watchlisted = false
    var blocklisted = false
}

enum IssueKind: String, CaseIterable, Sendable {
    case video, audio, subtitles, other

    var title: LText {
        switch self {
        case .video: "issue.kind.video"
        case .audio: "issue.kind.audio"
        case .subtitles: "issue.kind.subtitles"
        case .other: "issue.kind.other"
        }
    }
}

enum RequestFilter: String, CaseIterable, Sendable {
    case all, pending, approved, available, declined, failed

    var title: LText {
        switch self {
        case .all: "requests.tab.all"
        case .pending: "requests.tab.pending"
        case .approved: "requests.tab.approved"
        case .available: "requests.tab.available"
        case .declined: "requests.tab.declined"
        case .failed: "requests.tab.failed"
        }
    }
}

struct RequestCounts: Sendable, Equatable {
    var pending = 0, approved = 0, available = 0, declined = 0, failed = 0

    func count(_ filter: RequestFilter) -> Int {
        switch filter {
        case .all: pending + approved + available + declined + failed
        case .pending: pending
        case .approved: approved
        case .available: available
        case .declined: declined
        case .failed: failed
        }
    }
}

struct RequestPage: Sendable {
    var items: [RequestRecord]
    var total: Int
}
