import SwiftUI

/// Seasons of a show; a tap opens the episodes of that season (loaded on first open), as on the web.
struct SeasonList: View {
    let tmdbId: Int
    let seasons: [SeasonInfo]
    /// Seasons already covered by a request, with the state to show on them.
    let covered: Set<Int>
    let state: RequestState

    @Environment(\.appContext) private var context
    @State private var open: Int?
    @State private var episodes: [Int: [EpisodeInfo]] = [:]
    @State private var loading: Int?
    @State private var failed: [Int: String] = [:]

    var body: some View {
        VStack(alignment: .leading, spacing: Tokens.Spacing.md) {
            Text("seasons.title").font(.title3.weight(.bold)).accessibilityAddTraits(.isHeader)
            ForEach(seasons) { season in
                VStack(spacing: 0) {
                    Button { Task { await toggle(season.number) } } label: { header(season) }
                        .buttonStyle(.plain)
                        .accessibilityAddTraits(open == season.number ? .isSelected : [])
                    if open == season.number { episodeList(season.number) }
                }
                .background(Tokens.palette.card, in: .rect(cornerRadius: Tokens.Radius.md))
                .overlay { RoundedRectangle(cornerRadius: Tokens.Radius.md).strokeBorder(Tokens.palette.border) }
                .clipShape(.rect(cornerRadius: Tokens.Radius.md))
            }
        }
    }

    private func header(_ season: SeasonInfo) -> some View {
        HStack(spacing: Tokens.Spacing.md) {
            VStack(alignment: .leading, spacing: 2) {
                Text(verbatim: season.name).font(.body.weight(.semibold))
                Text(verbatim: meta(season)).font(.footnote).foregroundStyle(Tokens.palette.mutedFg)
            }
            Spacer()
            if covered.contains(season.number) { StatusBadge(state: state) }
            Image(systemName: "chevron.down")
                .font(.footnote.weight(.semibold))
                .foregroundStyle(Tokens.palette.mutedFg)
                .rotationEffect(.degrees(open == season.number ? 180 : 0))
        }
        .padding(Tokens.Spacing.md)
        .contentShape(.rect)
    }

    @ViewBuilder private func episodeList(_ number: Int) -> some View {
        VStack(alignment: .leading, spacing: Tokens.Spacing.md) {
            if loading == number {
                Text("seasons.loading").font(.footnote).foregroundStyle(Tokens.palette.mutedFg)
            } else if let message = failed[number] {
                Text(verbatim: message).font(.footnote).foregroundStyle(Tokens.palette.mutedFg)
            } else {
                ForEach(episodes[number] ?? []) { episode in EpisodeRow(episode: episode) }
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(Tokens.Spacing.md)
        .background(Tokens.palette.muted.opacity(0.4))
    }

    private func meta(_ season: SeasonInfo) -> String {
        let count = Plural.text("seasons.episodes", count: season.episodeCount)
        guard let year = season.airDate.flatMap({ $0.count >= 4 ? String($0.prefix(4)) : nil }) else { return count }
        return "\(count) · \(year)"
    }

    private func toggle(_ number: Int) async {
        if open == number { open = nil; return }
        open = number
        guard episodes[number] == nil, let api = context?.api else { return }
        loading = number
        failed[number] = nil
        defer { loading = nil }
        do { episodes[number] = try await api.episodes(tmdbId: tmdbId, season: number) } catch { failed[number] = APIError.from(error).localizedMessage }
    }
}

private struct EpisodeRow: View {
    let episode: EpisodeInfo

    var body: some View {
        HStack(alignment: .top, spacing: Tokens.Spacing.md) {
            RemoteImage(path: episode.stillPath, size: .w342) {
                Text("seasons.no_preview").font(.caption2).foregroundStyle(Tokens.palette.mutedFg)
            }
            .frame(width: 128, height: 72)
            .background(Tokens.palette.muted)
            .clipShape(.rect(cornerRadius: Tokens.Radius.sm))
            .overlay(alignment: .bottomLeading) {
                Text(verbatim: "E\(episode.number)")
                    .font(.caption2.weight(.semibold))
                    .foregroundStyle(.white)
                    .padding(.horizontal, 5).padding(.vertical, 2)
                    .background(.black.opacity(0.7), in: .rect(cornerRadius: 4))
                    .padding(4)
            }
            .accessibilityHidden(true)
            VStack(alignment: .leading, spacing: 2) {
                Text(verbatim: episode.name).font(.subheadline.weight(.semibold))
                if !info.isEmpty { Text(verbatim: info).font(.caption).foregroundStyle(Tokens.palette.mutedFg) }
                if let overview = episode.overview {
                    Text(verbatim: overview).font(.caption).foregroundStyle(Tokens.palette.mutedFg).lineLimit(3).padding(.top, 2)
                }
            }
        }
        .accessibilityElement(children: .combine)
    }

    private var info: String {
        var parts: [String] = []
        if let date = episode.airDate, let parsed = Self.parser.date(from: date) { parts.append(parsed.formatted(date: .abbreviated, time: .omitted)) }
        if let minutes = episode.runtimeMinutes, minutes > 0 { parts.append(Plural.text("time.min", count: minutes)) }
        if episode.voteAverage > 0 { parts.append("★ " + String(format: "%.1f", episode.voteAverage)) }
        return parts.joined(separator: " · ")
    }

    private static let parser: DateFormatter = {
        let f = DateFormatter()
        f.dateFormat = "yyyy-MM-dd"
        f.locale = Locale(identifier: "en_US_POSIX")
        return f
    }()
}
