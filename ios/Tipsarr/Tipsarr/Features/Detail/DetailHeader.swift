import SwiftUI

/// Backdrop, poster, title, year / runtime and the state badge.
/// The backdrop spans the whole width; the poster and title sit in the same centred column as the rest of the page.
struct DetailHeader: View {
    let detail: MediaDetail
    var badge: RequestState?
    /// Width of the page (a full screen, a split view column or a Duo pane); 0 until measured.
    var width: CGFloat = 0

    /// A column as wide as an iPad half screen or more: taller backdrop and a bigger poster.
    private var roomy: Bool { width >= 600 }
    /// Taller on wide pages so the banner is not a thin strip; never more than 400 pt.
    private var backdropHeight: CGFloat { roomy ? max(260, min(width * 0.38, 400)) : 260 }
    private var posterWidth: CGFloat { roomy ? 150 : 120 }
    /// How far the poster row rises into the backdrop.
    private var overlap: CGFloat { roomy ? 80 : 64 }

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            backdrop
            HStack(alignment: .bottom, spacing: Tokens.Spacing.lg) {
                RemoteImage(path: detail.posterPath, size: .w342) {
                    Image(systemName: "film").font(.title).foregroundStyle(Tokens.palette.mutedFg)
                }
                .frame(width: posterWidth, height: posterWidth * Tokens.Size.posterRatio)
                .background(Tokens.palette.muted)
                .clipShape(.rect(cornerRadius: Tokens.Radius.md))
                .overlay { RoundedRectangle(cornerRadius: Tokens.Radius.md).strokeBorder(Tokens.palette.border) }
                .shadow(color: .black.opacity(0.25), radius: 10, y: 4)
                .accessibilityHidden(true)
                VStack(alignment: .leading, spacing: Tokens.Spacing.xs) {
                    Text(verbatim: detail.title)
                        .font(roomy ? .title.weight(.bold) : .title2.weight(.bold))
                        .accessibilityAddTraits(.isHeader)
                    if !meta.isEmpty {
                        Text(verbatim: meta).font(.footnote).foregroundStyle(Tokens.palette.mutedFg)
                    }
                    if let badge { StatusBadge(state: badge) }
                }
                .frame(maxWidth: .infinity, alignment: .leading)
            }
            .padding(.horizontal, Tokens.Spacing.lg)
            .frame(maxWidth: MediaDetailView.columnWidth)
            .frame(maxWidth: .infinity)
            .offset(y: -overlap)
            .padding(.bottom, -overlap)
        }
    }

    /// Pulling the page down stretches the image instead of showing the edge of the banner.
    private var backdrop: some View {
        GeometryReader { proxy in
            let pull = max(0, proxy.frame(in: .scrollView).minY)
            RemoteImage(path: detail.backdropPath ?? detail.posterPath, size: .w780) {
                Tokens.palette.muted
            }
            .frame(width: proxy.size.width, height: backdropHeight + pull)
            .clipped()
            .overlay {
                LinearGradient(colors: [.clear, Tokens.palette.bg], startPoint: .center, endPoint: .bottom)
            }
            .offset(y: -pull)
        }
        .frame(height: backdropHeight)
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
            .readableColumn(MediaDetailView.columnWidth)
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
