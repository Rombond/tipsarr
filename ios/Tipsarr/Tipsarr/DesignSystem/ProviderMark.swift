import SwiftUI

/// Logo of a score provider (TMDB, IMDb, Rotten Tomatoes) or Metacritic's coloured square.
/// Logos come from Seerr (MIT), see `ios/RATING-LOGOS-NOTICE.md`.
struct ProviderMark: View {
    let source: RatingSource
    var value: Double?
    /// Height of the logo; the Metacritic square scales with it.
    var height: CGFloat = 14

    var body: some View {
        switch source {
        case .tmdb:
            // The logo has white letters: it needs TMDB's dark blue behind it on a light screen.
            logo("RatingTMDB", height: height * 0.6, name: "TMDB")
                .padding(.horizontal, height * 0.3)
                .padding(.vertical, height * 0.22)
                .background(Color(hex: 0x0D253FFF), in: .rect(cornerRadius: height * 0.25))
        case .imdb:
            logo("RatingIMDb", height: height, name: "IMDb")
        case .rottenTomatoes:
            logo((value ?? 100) >= 60 ? "RatingRTFresh" : "RatingRTRotten", height: height * 1.15, name: "Rotten Tomatoes")
        case .metacritic:
            MetacriticSquare(value: value, label: "MC", height: height)
        }
    }

    private func logo(_ asset: String, height: CGFloat, name: String) -> some View {
        Image(asset)
            .resizable()
            .scaledToFit()
            .frame(height: height)
            .accessibilityLabel(Text(verbatim: name))
    }
}

/// Metacritic's score in its own colour bands (green from 61, amber from 40, red below).
struct MetacriticSquare: View {
    var value: Double?
    /// Text to show instead of the score (e.g. "MC" above a score).
    var label: String?
    var height: CGFloat = 14

    private var color: Color {
        guard let value else { return Tokens.palette.mutedFg }
        return value >= 61 ? Tokens.Status.available : (value >= 40 ? Tokens.Status.requested : Tokens.palette.destructive)
    }

    var body: some View {
        Text(verbatim: label ?? value.map { String(Int($0.rounded())) } ?? "")
            .font(.system(size: height * 0.72, weight: .heavy))
            .monospacedDigit()
            .foregroundStyle(.white)
            .padding(.horizontal, height * 0.3)
            .frame(minWidth: height * 1.4, minHeight: height * 1.4)
            .background(color, in: .rect(cornerRadius: height * 0.25))
            .accessibilityLabel(Text(verbatim: "Metacritic"))
    }
}
