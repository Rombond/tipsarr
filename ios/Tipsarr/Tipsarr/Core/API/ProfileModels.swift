import Foundation

enum RatingSource: String, CaseIterable, Sendable {
    case tmdb, imdb, metacritic, rottenTomatoes
}

struct DeviceSession: Sendable, Identifiable, Hashable {
    enum Platform: String, Sendable { case web, ios, android }
    var id: String
    var platform: Platform
    var deviceName: String
    var appVersion: String
    var lastSeen: Date
    var current: Bool
}
