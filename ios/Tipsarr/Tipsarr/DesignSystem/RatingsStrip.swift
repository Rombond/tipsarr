import SwiftUI

/// One row of scores between the poster and the request button.
/// Movies: TMDB, IMDb, Rotten Tomatoes, Metacritic. TV: TMDB only.
struct RatingsStrip: View {
    let ratings: RatingsSummary

    private struct Entry: Identifiable {
        let id: String
        let value: String
        let hint: LocalizedStringResource?
    }

    private var entries: [Entry] {
        var list: [Entry] = []
        if let tmdb = ratings.tmdb { list.append(Entry(id: "TMDB", value: tmdb.formatted(.number.precision(.fractionLength(1))), hint: "card.rating_hint")) }
        if let imdb = ratings.imdb { list.append(Entry(id: "IMDb", value: imdb.formatted(.number.precision(.fractionLength(1))), hint: nil)) }
        if let rt = ratings.rottenTomatoes { list.append(Entry(id: "RT", value: "\(Int(rt.rounded()))%", hint: "ratings.rt_hint")) }
        if let mc = ratings.metacritic { list.append(Entry(id: "Metacritic", value: "\(Int(mc.rounded()))", hint: "ratings.mc_hint")) }
        return list
    }

    var body: some View {
        if !entries.isEmpty {
            HStack(spacing: 0) {
                ForEach(Array(entries.enumerated()), id: \.element.id) { index, entry in
                    if index > 0 { Divider().frame(height: 28) }
                    VStack(spacing: Tokens.Spacing.xs) {
                        Text(verbatim: entry.id)
                            .font(.caption2.weight(.semibold))
                            .foregroundStyle(Tokens.palette.mutedFg)
                            .textCase(nil)
                        Text(verbatim: entry.value).font(.headline.weight(.bold))
                    }
                    .frame(maxWidth: .infinity)
                    .accessibilityElement(children: .combine)
                }
            }
            .padding(.vertical, Tokens.Spacing.sm)
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
