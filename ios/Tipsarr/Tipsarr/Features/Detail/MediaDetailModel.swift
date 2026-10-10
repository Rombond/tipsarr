import Foundation
import Observation

/// What the main button of the detail screen does.
enum DetailAction: Equatable {
    case request
    case inProgress(RequestState, cancellable: Bool)
    case available(URL?)
    case requestAgain
    case failed(canRetry: Bool)
}

@MainActor @Observable
final class MediaDetailModel {
    enum Phase: Equatable { case loading, loaded, failed(APIError) }

    let route: MediaRoute
    private let context: AppContext

    private(set) var phase: Phase = .loading
    private(set) var detail: MediaDetail?
    private(set) var ratings = RatingsSummary()
    private(set) var flags = TitleFlags()
    /// The signed-in user's newest request for this title.
    private(set) var request: RequestRecord?
    /// Seasons already covered by one of the user's requests (open or available).
    private(set) var coveredSeasons: Set<Int> = []
    private(set) var options: RequestOptions?
    private(set) var busy = false

    init(route: MediaRoute, context: AppContext) {
        self.route = route
        self.context = context
    }

    var isAdmin: Bool { context.profile.isAdmin }
    var canChooseFolder: Bool { context.canChooseFolder }

    // MARK: Loading

    func load() async {
        if detail == nil { phase = .loading }
        do {
            let loaded = try await context.api.detail(route.type, id: route.tmdbId)
            detail = loaded
            ratings.tmdb = loaded.voteAverage > 0 ? loaded.voteAverage : nil
            phase = .loaded
        } catch {
            if detail == nil { phase = .failed(APIError.from(error)) }
            return
        }
        // The rest is optional: the screen already works without it.
        async let scores = movieScores()
        async let titleFlags = try? context.api.flags(route.type, id: route.tmdbId)
        async let mine = try? context.api.myRequests()
        if let scores = await scores { ratings.imdb = scores.imdb; ratings.rottenTomatoes = scores.rottenTomatoes; ratings.metacritic = scores.metacritic; ratings.rottenTomatoesURL = scores.rottenTomatoesURL }
        if let titleFlags = await titleFlags { flags = titleFlags }
        if let mine = await mine { apply(requests: mine) }
    }

    /// TV scores other than TMDB need Radarr data that only exists for movies.
    private func movieScores() async -> RatingsSummary? {
        guard route.type == .movie else { return nil }
        return try? await context.api.ratings(.movie, id: route.tmdbId)
    }

    private func apply(requests all: [RequestRecord]) {
        let forTitle = all.filter { $0.type == route.type && $0.tmdbId == route.tmdbId }
        request = forTitle.max { $0.createdAt < $1.createdAt }
        coveredSeasons = Set(forTitle.filter { $0.state != .declined && $0.state != .failed }.flatMap(\.seasons))
    }

    // MARK: State of the main button

    var action: DetailAction {
        guard let detail else { return .request }
        if detail.availability == .available { return .available(detail.watchURL) }
        if let request {
            if request.isOpen { return .inProgress(request.state, cancellable: true) }
            if request.state == .declined { return .requestAgain }
            if request.state == .failed { return .failed(canRetry: isAdmin) }
        }
        if detail.requestOpen { return .inProgress(.requested, cancellable: false) }
        return .request
    }

    /// The badge next to the title.
    var badge: RequestState? {
        switch action {
        case .available: .available
        case .inProgress(let state, _): state
        case .requestAgain: .declined
        case .failed: .failed
        case .request: detail?.availability == .partial ? .partial : nil
        }
    }

    var regularSeasons: [SeasonInfo] { (detail?.seasons ?? []).filter { $0.number > 0 } }

    // MARK: Actions

    /// Loads quality profiles. Returns whether the request needs a sheet (a choice to make).
    func prepareRequest() async -> Bool {
        busy = true
        defer { busy = false }
        options = try? await context.api.requestOptions(route.type)
        return options != nil || (route.type == .tv && regularSeasons.count > 1)
    }

    @discardableResult
    func submitRequest(seasons: [Int]?, profileID: Int?, folder: String?) async throws -> RequestRecord {
        busy = true
        defer { busy = false }
        let record = try await context.api.createRequest(route.type, tmdbId: route.tmdbId, seasons: seasons, profileID: profileID, folder: folder)
        request = record
        coveredSeasons.formUnion(record.seasons)
        detail?.requestOpen = true
        return record
    }

    func cancelRequest() async throws {
        guard let request else { return }
        busy = true
        defer { busy = false }
        try await context.api.deleteRequest(id: request.id)
        self.request = nil
        coveredSeasons.subtract(request.seasons)
        detail?.requestOpen = false
    }

    func retryRequest() async throws {
        guard let request else { return }
        busy = true
        defer { busy = false }
        self.request = try await context.api.retryRequest(id: request.id)
    }

    func toggleWatchlist() async throws {
        let on = !flags.watchlisted
        try await context.api.setWatchlisted(on, route.type, id: route.tmdbId)
        flags.watchlisted = on
    }

    func toggleHidden() async throws {
        let on = !flags.blocklisted
        try await context.api.setBlocklisted(on, route.type, id: route.tmdbId)
        flags.blocklisted = on
    }

    func report(kind: IssueKind, message: String) async throws {
        try await context.api.reportIssue(route.type, tmdbId: route.tmdbId, kind: kind, message: message)
    }
}
