import Foundation

/// Lean copies of what the app needs from the generated schemas, so screens
/// never depend on generated types.
struct ServerStatus: Sendable, Equatable {
    var version: String
    var apiVersion: Int
    var minAppVersion: String
    var userFolderChoice: Bool
    var defaultLanguage: String
    var pushAvailable: Bool
}

struct Profile: Sendable, Equatable, Codable {
    var id: String
    var name: String
    var role: String
    var region: String
    var language: String
    var ratingSource: String
    var createdAt: Date?
    var lastLoginAt: Date?

    var isAdmin: Bool { role == "admin" }
}

struct SignInResult: Sendable {
    var token: String
    var profile: Profile
}

/// A server the user connected to, with the status read at that moment.
struct Server: Sendable, Equatable {
    var url: URL
    var status: ServerStatus

    var host: String { url.host() ?? url.absoluteString }
}

enum MediaType: String, Sendable, Hashable, Codable {
    case movie, tv
}

/// A title in a list (discover, search, suggestions).
struct MediaItem: Sendable, Hashable, Identifiable {
    var type: MediaType
    var tmdbId: Int
    var title: String
    var posterPath: String?
    var releaseYear: String?
    var voteAverage: Double
    var state: RequestState?

    var id: String { "\(type.rawValue)-\(tmdbId)" }
    var route: MediaRoute { MediaRoute(type: type, tmdbId: tmdbId, title: title) }
}

/// Navigation value for the media detail screen (step 4).
struct MediaRoute: Hashable, Sendable {
    var type: MediaType
    var tmdbId: Int
    var title: String
}

struct MediaPage: Sendable {
    var items: [MediaItem]
    var page: Int
    var totalPages: Int
    var hasMore: Bool { page < totalPages }
}

struct SuggestionRow: Sendable, Identifiable {
    var id: String
    var title: String
    var items: [MediaItem]
}

struct SuggestionResult: Sendable {
    var rows: [SuggestionRow]
    /// The server is still building the recommendations.
    var generating: Bool
}

struct Genre: Sendable, Hashable, Identifiable {
    var id: Int
    var name: String
}

struct PersonSummary: Sendable, Hashable, Identifiable {
    var id: Int
    var name: String
    var department: String?
    var profilePath: String?
    var route: PersonRoute { PersonRoute(id: id, name: name) }
}

struct PersonRoute: Hashable, Sendable {
    var id: Int
    var name: String
}

struct GenreRoute: Hashable, Sendable {
    var type: MediaType
    var id: Int
    var name: String
}

struct PersonDetail: Sendable {
    var id: Int
    var name: String
    var department: String?
    var biography: String?
    var birthday: String?
    var birthplace: String?
    var deathday: String?
    var profilePath: String?
    var credits: [MediaItem]
}

struct SearchPage: Sendable {
    var items: [MediaItem]
    var people: [PersonSummary]
    var page: Int
    var totalPages: Int
    var hasMore: Bool { page < totalPages }
}
