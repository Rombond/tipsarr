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
