import SwiftUI

/// Poster with title, year and request state. The image is loaded with the
/// active account's token (see `RemoteImage`).
struct PosterCard: View {
    let title: String
    var subtitle: String?
    var posterPath: String?
    var posterSize: TMDBSize = .w342
    /// Poster served by the Tipsarr server itself (library titles come from Jellyfin); wins over `posterPath`.
    var serverPosterPath: String?
    var watched = false
    var rating: RatingInput?
    var state: RequestState?

    var body: some View {
        VStack(alignment: .leading, spacing: Tokens.Spacing.sm) {
            poster
                .aspectRatio(1 / Tokens.Size.posterRatio, contentMode: .fit)
                .overlay(alignment: .topTrailing) {
                    if let state {
                        StatusBadge(state: state, compact: true).padding(Tokens.Spacing.sm)
                    }
                }
                .overlay(alignment: .topLeading) {
                    if watched {
                        Image(systemName: "eye.fill")
                            .font(.caption2)
                            .foregroundStyle(.white)
                            .padding(Tokens.Spacing.xs + 2)
                            .background(.black.opacity(0.55), in: .circle)
                            .padding(Tokens.Spacing.sm)
                    }
                }
                .clipShape(.rect(cornerRadius: Tokens.Radius.md))
            Text(title)
                .font(.subheadline.weight(.medium))
                .lineLimit(1)
                .foregroundStyle(Tokens.palette.fg)
            if subtitle != nil || rating != nil {
                HStack(spacing: Tokens.Spacing.xs) {
                    if let subtitle {
                        Text(subtitle)
                            .font(.footnote)
                            .foregroundStyle(Tokens.palette.mutedFg)
                            .lineLimit(1)
                    }
                    Spacer(minLength: 0)
                    if let rating { RatingLabel(input: rating) }
                }
            }
        }
        .frame(maxWidth: .infinity, alignment: .topLeading)
        .contentShape(.hoverEffect, .rect(cornerRadius: Tokens.Radius.md))
        .hoverEffect(.lift)
        .accessibilityElement(children: .ignore)
        .accessibilityLabel(accessibilityText)
    }

    init(title: String, subtitle: String? = nil, posterPath: String? = nil, posterSize: TMDBSize = .w342,
         serverPosterPath: String? = nil, watched: Bool = false, rating: RatingInput? = nil, state: RequestState? = nil) {
        self.title = title
        self.subtitle = subtitle
        self.posterPath = posterPath
        self.posterSize = posterSize
        self.serverPosterPath = serverPosterPath
        self.watched = watched
        self.rating = rating
        self.state = state
    }

    /// Standard card for a list item: year, score and request state.
    init(item: MediaItem, posterSize: TMDBSize = .w342) {
        self.init(title: item.title, subtitle: item.releaseYear, posterPath: item.posterPath, posterSize: posterSize,
                  rating: RatingInput(type: item.type, tmdbId: item.tmdbId, tmdb: item.voteAverage), state: item.state)
    }

    private var poster: some View {
        Group {
            if let serverPosterPath {
                RemoteImage(serverPath: serverPosterPath) { placeholder }
            } else {
                RemoteImage(path: posterPath, size: posterSize) { placeholder }
            }
        }
        .background(Tokens.palette.muted)
    }

    private var placeholder: some View {
        ZStack {
            Tokens.palette.muted
            Image(systemName: "film")
                .font(.title2)
                .foregroundStyle(Tokens.palette.mutedFg)
        }
    }

    private var accessibilityText: Text {
        var text = Text(verbatim: title)
        if let subtitle { text = text + Text(verbatim: ", \(subtitle)") }
        if let state { text = text + Text(verbatim: ", ") + Text(state.title) }
        return text
    }
}

#Preview {
    LazyVGrid(columns: Array(repeating: GridItem(.flexible(), spacing: Tokens.Spacing.md), count: 3), spacing: Tokens.Spacing.lg) {
        PosterCard(title: "Blade Runner 2049", subtitle: "2017", state: .available)
        PosterCard(title: "Severance", subtitle: "2022", state: .requested)
        PosterCard(title: "A very long title that gets truncated", subtitle: "1999")
    }
    .padding()
}
