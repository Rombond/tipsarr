import SwiftUI

/// One row of scores between the poster and the request button, each with its provider's logo.
/// Movies: TMDB, IMDb, Rotten Tomatoes, Metacritic. TV: TMDB only.
struct RatingsStrip: View {
    let ratings: RatingsSummary

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

    var body: some View {
        if !entries.isEmpty {
            HStack(spacing: 0) {
                ForEach(Array(entries.enumerated()), id: \.element.id) { index, entry in
                    if index > 0 { Divider().frame(height: 28) }
                    HStack(spacing: Tokens.Spacing.sm) {
                        if entry.source == .metacritic {
                            MetacriticSquare(value: entry.value, height: 18)
                        } else {
                            ProviderMark(source: entry.source, value: entry.value, height: 18)
                            Text(verbatim: RatingProvider.format(entry.value, as: entry.source) ?? "").font(.headline.weight(.bold))
                        }
                    }
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
