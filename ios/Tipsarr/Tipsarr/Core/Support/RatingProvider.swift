import SwiftUI

/// What a card needs to show a score.
struct RatingInput: Hashable, Sendable {
    var type: MediaType
    var tmdbId: Int
    /// The TMDB (or library) score the list already carries, 0-10.
    var tmdb: Double?
}

/// Scores for posters, following the "Score on posters" preference.
/// TMDB comes with every list. IMDb, Metacritic and Rotten Tomatoes are fetched in batches for movies
/// (shows only have TMDB scores, so they keep showing it).
@MainActor @Observable
final class RatingProvider {
    private let api: TipsarrAPI
    private(set) var source: RatingSource
    private var scores: [Int: Double] = [:]
    @ObservationIgnored private var asked: Set<Int> = []
    @ObservationIgnored private var pending: Set<Int> = []
    @ObservationIgnored private var flush: Task<Void, Never>?

    init(api: TipsarrAPI, source: RatingSource) {
        self.api = api
        self.source = source
    }

    func setSource(_ new: RatingSource) {
        guard new != source else { return }
        source = new
        scores = [:]
        asked = []
        pending = []
        flush?.cancel()
    }

    struct Reading: Equatable {
        var source: RatingSource
        var value: Double
        var text: String
    }

    /// The score to show on a card: the chosen source for movies, TMDB for shows. Nil while unknown.
    func reading(for input: RatingInput) -> Reading? {
        let (shown, value): (RatingSource, Double?) = (source == .tmdb || input.type == .tv) ? (.tmdb, input.tmdb) : (source, scores[input.tmdbId])
        guard let value, let text = Self.format(value, as: shown) else { return nil }
        return Reading(source: shown, value: value, text: text)
    }

    /// Asks for a movie's score; requests are grouped so one screen makes one or two calls.
    func request(_ input: RatingInput) {
        guard source != .tmdb, input.type == .movie, !asked.contains(input.tmdbId) else { return }
        asked.insert(input.tmdbId)
        pending.insert(input.tmdbId)
        flush?.cancel()
        flush = Task {
            try? await Task.sleep(for: .milliseconds(200))
            guard !Task.isCancelled else { return }
            await send()
        }
    }

    private func send() async {
        let current = source
        let ids = Array(pending)
        pending = []
        // The `ids` parameter is limited to 400 characters: about 40 ids per call.
        for chunk in stride(from: 0, to: ids.count, by: 40).map({ Array(ids[$0..<min($0 + 40, ids.count)]) }) {
            guard let found = try? await api.movieScores(chunk, source: current), current == source else { continue }
            for (id, summary) in found {
                let value: Double? = switch current {
                case .imdb: summary.imdb
                case .metacritic: summary.metacritic
                case .rottenTomatoes: summary.rottenTomatoes
                case .tmdb: summary.tmdb
                }
                if let value { scores[id] = value }
            }
        }
    }

    static func format(_ value: Double?, as source: RatingSource) -> String? {
        guard let value, value > 0 else { return nil }
        switch source {
        case .tmdb, .imdb: return value.formatted(.number.precision(.fractionLength(1)))
        case .metacritic: return String(Int(value.rounded()))
        case .rottenTomatoes: return "\(Int(value.rounded()))%"
        }
    }
}

private struct RatingProviderKey: EnvironmentKey {
    static let defaultValue: RatingProvider? = nil
}

extension EnvironmentValues {
    var ratingProvider: RatingProvider? {
        get { self[RatingProviderKey.self] }
        set { self[RatingProviderKey.self] = newValue }
    }
}

/// Provider logo and score, e.g. the IMDb logo and "7.8". Empty while there is no score.
struct RatingLabel: View {
    let input: RatingInput
    @Environment(\.ratingProvider) private var provider

    private var reading: RatingProvider.Reading? {
        provider?.reading(for: input)
            ?? RatingProvider.format(input.tmdb, as: .tmdb).map { .init(source: .tmdb, value: input.tmdb ?? 0, text: $0) }
    }

    var body: some View {
        Group {
            if let reading {
                HStack(spacing: Tokens.Spacing.xs) {
                    if reading.source == .metacritic {
                        MetacriticSquare(value: reading.value, height: 12)
                    } else {
                        ProviderMark(source: reading.source, value: reading.value, height: 12)
                        Text(verbatim: reading.text)
                            .font(.footnote.weight(.medium))
                            .foregroundStyle(Tokens.palette.mutedFg)
                    }
                }
                .accessibilityElement(children: .ignore)
                .accessibilityLabel(Text(verbatim: reading.text))
            }
        }
        .task(id: "\(input.type.rawValue)-\(input.tmdbId)-\(provider?.source.rawValue ?? "")") { provider?.request(input) }
    }
}
