import Foundation

struct UserStats: Sendable, Equatable {
    var requests = 0, movies = 0, shows = 0, pending = 0, approved = 0
    var available = 0, declined = 0, failed = 0, watchlist = 0, watched = 0
}

struct UserDetail: Sendable {
    var profile: Profile
    var stats: UserStats
}

struct JobStatus: Sendable, Identifiable, Hashable {
    var name: String
    var running: Bool
    var status: String
    var message: String
    var lastStarted: Date?
    var lastFinished: Date?
    var everySeconds: Int
    var id: String { name }
}

struct SyncStatus: Sendable {
    var canSync: Bool
    var jobs: [JobStatus]
    var movies: Int
    var shows: Int
}

enum StatsPeriod: String, CaseIterable, Sendable {
    case days30 = "30d", months12 = "12m", all

    var title: LText {
        switch self {
        case .days30: "stats.period_30d"
        case .months12: "stats.period_12m"
        case .all: "stats.period_all"
        }
    }
}

struct StatsBucket: Sendable, Identifiable, Hashable {
    var name: String
    var hours: Double
    var titles: Int
    var id: String { name }
}

struct StatsMonth: Sendable, Identifiable, Hashable {
    var month: String
    var hours: Double
    var plays: Int
    var id: String { month }
}

struct StatsTop: Sendable, Identifiable, Hashable {
    var type: MediaType
    var tmdbId: Int
    var title: String
    var year: Int?
    var plays: Int
    var hours: Double
    var posterPath: String?
    var id: String { "\(type.rawValue)-\(tmdbId)" }
    var route: MediaRoute { MediaRoute(type: type, tmdbId: tmdbId, title: title) }
}

struct StatsReport: Sendable {
    var hours: Double
    var plays: Int
    var titles: Int
    var movies: Int
    var shows: Int
    var requestsMade: Int
    var requestsAvailable: Int
    var requestsDeclined: Int
    var genres: [StatsBucket]
    var decades: [StatsBucket]
    var months: [StatsMonth]
    var weekdays: [Double]
    var hoursOfDay: [Double]
    var topMovies: [StatsTop]
    var topShows: [StatsTop]
    /// True when the numbers come from Jellyfin's Playback Reporting plugin; false means estimated.
    var exact: Bool
    var pluginHint: Bool
}
