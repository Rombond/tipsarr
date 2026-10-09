import SwiftUI

/// Backdrop, poster, title, year / runtime and the state badge.
struct DetailHeader: View {
    let detail: MediaDetail
    var badge: RequestState?

    private let backdropHeight: CGFloat = 260

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            backdrop
            HStack(alignment: .bottom, spacing: Tokens.Spacing.lg) {
                RemoteImage(path: detail.posterPath, size: .w342) {
                    Image(systemName: "film").font(.title).foregroundStyle(Tokens.palette.mutedFg)
                }
                .frame(width: 120, height: 120 * Tokens.Size.posterRatio)
                .background(Tokens.palette.muted)
                .clipShape(.rect(cornerRadius: Tokens.Radius.md))
                .overlay { RoundedRectangle(cornerRadius: Tokens.Radius.md).strokeBorder(Tokens.palette.border) }
                .shadow(color: .black.opacity(0.25), radius: 10, y: 4)
                .accessibilityHidden(true)
                VStack(alignment: .leading, spacing: Tokens.Spacing.sm) {
                    Text(verbatim: detail.title)
                        .font(.title2.weight(.bold))
                        .accessibilityAddTraits(.isHeader)
                    if !meta.isEmpty {
                        Text(verbatim: meta).font(.footnote).foregroundStyle(Tokens.palette.mutedFg)
                    }
                    if let badge { StatusBadge(state: badge) }
                }
                .frame(maxWidth: .infinity, alignment: .leading)
            }
            .padding(.horizontal, Tokens.Spacing.lg)
            .offset(y: -64)
            .padding(.bottom, -64)
        }
    }

    private var backdrop: some View {
        RemoteImage(path: detail.backdropPath ?? detail.posterPath, size: .w780) {
            Tokens.palette.muted
        }
        .frame(height: backdropHeight)
        .frame(maxWidth: .infinity)
        .clipped()
        .overlay {
            LinearGradient(colors: [.clear, Tokens.palette.bg], startPoint: .center, endPoint: .bottom)
        }
        .accessibilityHidden(true)
    }

    /// "2024 · 2h 46m" for movies, "2022 · 3 seasons" for shows.
    private var meta: String {
        var parts: [String] = []
        if let year = detail.year { parts.append(year) }
        if detail.type == .tv {
            if let seasons = detail.numberOfSeasons, seasons > 0 { parts.append(Plural.text("m.detail.seasons_count", count: seasons)) }
        } else if let minutes = detail.runtimeMinutes, minutes > 0 {
            parts.append(Duration.seconds(minutes * 60).formatted(.units(allowed: [.hours, .minutes], width: .narrow)))
        }
        return parts.joined(separator: " · ")
    }
}

struct DetailSkeleton: View {
    let title: String

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: Tokens.Spacing.xl) {
                Color.clear.frame(height: 200)
                HStack(alignment: .bottom, spacing: Tokens.Spacing.lg) {
                    RoundedRectangle(cornerRadius: Tokens.Radius.md).fill(Tokens.palette.border)
                        .frame(width: 120, height: 120 * Tokens.Size.posterRatio)
                    VStack(alignment: .leading, spacing: Tokens.Spacing.sm) {
                        Text(verbatim: title).font(.title2.weight(.bold))
                        SkeletonBlock(height: 12).frame(width: 90)
                    }
                    .frame(maxWidth: .infinity, alignment: .leading)
                }
                SkeletonBlock(height: 48, cornerRadius: Tokens.Radius.md)
                SkeletonBlock(height: 50, cornerRadius: Tokens.Radius.md)
                ForEach(0..<4, id: \.self) { _ in SkeletonBlock(height: 14) }
            }
            .padding(.horizontal, Tokens.Spacing.lg)
            .skeleton()
        }
    }
}

/// Wraps its children onto new lines (genre pills).
struct FlowLayout: Layout {
    var spacing: CGFloat = 8

    func sizeThatFits(proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) -> CGSize {
        arrange(width: proposal.width ?? .infinity, subviews: subviews).size
    }

    func placeSubviews(in bounds: CGRect, proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) {
        let result = arrange(width: bounds.width, subviews: subviews)
        for (index, frame) in result.frames.enumerated() {
            subviews[index].place(at: CGPoint(x: bounds.minX + frame.minX, y: bounds.minY + frame.minY), proposal: .unspecified)
        }
    }

    private func arrange(width: CGFloat, subviews: Subviews) -> (size: CGSize, frames: [CGRect]) {
        var frames: [CGRect] = []
        var x: CGFloat = 0, y: CGFloat = 0, rowHeight: CGFloat = 0, maxX: CGFloat = 0
        for subview in subviews {
            let size = subview.sizeThatFits(.unspecified)
            if x + size.width > width, x > 0 {
                x = 0
                y += rowHeight + spacing
                rowHeight = 0
            }
            frames.append(CGRect(origin: CGPoint(x: x, y: y), size: size))
            x += size.width + spacing
            rowHeight = max(rowHeight, size.height)
            maxX = max(maxX, x - spacing)
        }
        return (CGSize(width: maxX, height: y + rowHeight), frames)
    }
}
