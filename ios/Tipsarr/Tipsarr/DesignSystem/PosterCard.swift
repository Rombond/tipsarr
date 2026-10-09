import SwiftUI

/// Poster with title, year and request state. Image loading goes through
/// `AsyncImage` for now; Core/Images swaps in the bearer-aware loader.
struct PosterCard: View {
    let title: String
    var subtitle: String?
    var imageURL: URL?
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
                .clipShape(.rect(cornerRadius: Tokens.Radius.md))
            Text(title)
                .font(.subheadline.weight(.medium))
                .lineLimit(1)
                .foregroundStyle(Tokens.palette.fg)
            if let subtitle {
                Text(subtitle)
                    .font(.footnote)
                    .foregroundStyle(Tokens.palette.mutedFg)
                    .lineLimit(1)
            }
        }
        .frame(maxWidth: .infinity, alignment: .topLeading)
        .accessibilityElement(children: .ignore)
        .accessibilityLabel(accessibilityText)
    }

    private var poster: some View {
        AsyncImage(url: imageURL) { phase in
            switch phase {
            case .success(let image):
                image.resizable().scaledToFill()
            default:
                ZStack {
                    Tokens.palette.muted
                    Image(systemName: "film")
                        .font(.title2)
                        .foregroundStyle(Tokens.palette.mutedFg)
                }
            }
        }
        .background(Tokens.palette.muted)
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
