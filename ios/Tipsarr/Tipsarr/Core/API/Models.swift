import Foundation

/// Lean copies of what the app needs from the generated schemas, so screens
/// never depend on generated types.
struct ServerStatus: Sendable, Equatable {
    var version: String
    var apiVersion: Int
    var minAppVersion: String
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
    /// What the person may do: the sum of their permission flags (everything for an admin). 0 until the server says.
    var effectivePermissions: Int64 = 0
    /// The person's own stored mask and request limits, only filled by the admin user list.
    var permissions: Int64 = 0
    var quota: UserQuota = .init()

    /// The role: admins can do everything. For what a person may do use `can`.
    var isAdmin: Bool { role == "admin" }

    func can(_ flags: Permission...) -> Bool { Permission(rawValue: effectivePermissions).contains(flags) }
    func canAny(_ flags: Permission...) -> Bool { flags.contains { can($0) } }
    func mayRequest(_ type: MediaType) -> Bool { can(.request) || can(type == .movie ? .requestMovie : .requestTV) }
}

extension Profile {
    private enum CodingKeys: String, CodingKey {
        case id, name, role, region, language, ratingSource, createdAt, lastLoginAt, effectivePermissions, permissions, quota
    }

    /// Accounts saved before permissions existed have no `effectivePermissions`: they decode as 0 and the app fetches the profile again.
    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        self.init(id: try c.decode(String.self, forKey: .id), name: try c.decode(String.self, forKey: .name),
                  role: try c.decode(String.self, forKey: .role), region: try c.decode(String.self, forKey: .region),
                  language: try c.decode(String.self, forKey: .language), ratingSource: try c.decode(String.self, forKey: .ratingSource),
                  createdAt: try c.decodeIfPresent(Date.self, forKey: .createdAt), lastLoginAt: try c.decodeIfPresent(Date.self, forKey: .lastLoginAt),
                  effectivePermissions: try c.decodeIfPresent(Int64.self, forKey: .effectivePermissions) ?? 0,
                  permissions: try c.decodeIfPresent(Int64.self, forKey: .permissions) ?? 0,
                  quota: try c.decodeIfPresent(UserQuota.self, forKey: .quota) ?? .init())
    }
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
    var overview: String?
    var backdropPath: String?
    /// TMDB genre ids (movie or show list, by `type`); used by the search filters.
    var genreIds: [Int] = []

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

/// One genre name with its TMDB id in the movie list and/or in the show list (the two lists differ a little).
struct MergedGenre: Hashable, Sendable, Identifiable {
    var name: String
    var movieID: Int?
    var tvID: Int?
    var id: String { name }

    /// The two genre lists as one, sorted by name; a name present in both keeps both ids.
    static func merge(movies: [Genre], shows: [Genre]) -> [MergedGenre] {
        var byName: [String: MergedGenre] = [:]
        for genre in movies { byName[genre.name, default: MergedGenre(name: genre.name)].movieID = genre.id }
        for genre in shows { byName[genre.name, default: MergedGenre(name: genre.name)].tvID = genre.id }
        return byName.values.sorted { $0.name.localizedStandardCompare($1.name) == .orderedAscending }
    }

    func id(for type: MediaType) -> Int? { type == .movie ? movieID : tvID }
    var route: GenreRoute { GenreRoute(name: name, movieID: movieID, tvID: tvID) }
}

struct GenreRoute: Hashable, Sendable {
    var name: String
    var movieID: Int?
    var tvID: Int?
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
