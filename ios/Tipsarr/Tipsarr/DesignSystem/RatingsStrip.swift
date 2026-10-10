import SwiftUI

/// One row of scores between the poster and the request button, each with its provider's logo.
/// Movies: TMDB, IMDb, Rotten Tomatoes, Metacritic. TV: TMDB only.
struct RatingsStrip: View {
    let ratings: RatingsSummary
    /// Where each score leads, as on the web: nil = not tappable (previews).
    var link: Link?

    /// What identifies the title on the score providers' sites.
    struct Link {
        var type: MediaType
        var tmdbId: Int
        var title: String
        var imdbId: String?

        /// The provider's page: TMDB and IMDb by id, Rotten Tomatoes' own page when known, else a search.
        func url(for source: RatingSource, rottenTomatoes: URL?) -> URL? {
            let q = title.addingPercentEncoding(withAllowedCharacters: .urlPathAllowed) ?? title
            switch source {
            case .tmdb: return URL(string: "https://www.themoviedb.org/\(type.rawValue)/\(tmdbId)")
            case .imdb: return URL(string: imdbId.map { "https://www.imdb.com/title/\($0)" } ?? "https://www.imdb.com/find/?q=\(q)")
            case .rottenTomatoes: return rottenTomatoes ?? URL(string: "https://www.rottentomatoes.com/search?search=\(q)")
            case .metacritic: return URL(string: "https://www.metacritic.com/search/\(q)/")
            }
        }
    }

    private struct Entry: Identifiable {
        let source: RatingSource
        let value: Double
        var id: RatingSource { source }
    }

    private var entries: [Entry] {
        var list: [Entry] = []
        if let value = ratings.tmdb { list.append(Entry(source: .tmdb, value: value)) }
        if let value = ratings.imdb { list.append(Entry(source: .imdb, value: value)) }
        if let value = ratings.rottenTomatoes { list.append(Entry(source: .rottenTomatoes, value: value)) }
        if let value = ratings.metacritic { list.append(Entry(source: .metacritic, value: value)) }
        return list
    }

    @ViewBuilder private func scoreView(_ entry: Entry) -> some View {
        let content = HStack(spacing: Tokens.Spacing.sm) {
            if entry.source == .metacritic {
                MetacriticSquare(value: entry.value, height: 18)
            } else {
                ProviderMark(source: entry.source, value: entry.value, height: 18)
                Text(verbatim: RatingProvider.format(entry.value, as: entry.source) ?? "").font(.headline.weight(.bold))
            }
        }
        .frame(minHeight: Tokens.Size.touchTarget - Tokens.Spacing.md)
        .padding(.horizontal, Tokens.Spacing.sm)
        .contentShape(.rect)
        if let url = link?.url(for: entry.source, rottenTomatoes: ratings.rottenTomatoesURL) {
            SwiftUI.Link(destination: url) { content }.buttonStyle(.plain)
        } else {
            content
        }
    }

    var body: some View {
        if !entries.isEmpty {
            HStack(spacing: 0) {
                ForEach(Array(entries.enumerated()), id: \.element.id) { index, entry in
                    if index > 0 { Divider().frame(height: 28) }
                    scoreView(entry)
                        .frame(maxWidth: .infinity)
                        .accessibilityElement(children: .combine)
                }
            }
            .padding(.vertical, Tokens.Spacing.md)
            .background(Tokens.palette.muted, in: .rect(cornerRadius: Tokens.Radius.md))
        }
    }
}

#Preview {
    VStack {
        RatingsStrip(ratings: .init(tmdb: 8.2, imdb: 8.5, rottenTomatoes: 92, metacritic: 79))
        RatingsStrip(ratings: .init(tmdb: 8.7))
    }
    .padding()
}
