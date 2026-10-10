import Observation
import SwiftUI

/// One weekend (or week) of the box office in a region, as the server stores it.
struct BoxOfficeChart: Sendable {
    struct Week: Sendable, Hashable { var key: String; var label: String }
    var region: String
    var regions: [String]
    var week: String
    var start: String
    var end: String
    var weeks: [Week]
    var entries: [BoxOfficeEntry]
}

struct BoxOfficeEntry: Sendable, Identifiable {
    var position: Int
    var title: String
    var weekendGross: Int
    var totalGross: Int
    var weeksInRelease: Int
    /// The matching TMDB title; nil when the chart title was not matched.
    var item: MediaItem?
    var id: Int { position }
}

extension TipsarrAPI {
    func boxOffice(region: String? = nil, week: String? = nil) async throws -> BoxOfficeChart {
        try await load { client in
            guard case .ok(let ok) = try await client.boxOffice(query: .init(region: region, week: week)) else { throw APIError.unexpected }
            let body = try ok.body.json
            return BoxOfficeChart(
                region: body.region, regions: body.regions, week: body.week, start: body.start ?? "", end: body.end ?? "",
                weeks: body.weeks.map { .init(key: $0.key, label: $0.label ?? $0.key) },
                entries: body.entries.map {
                    BoxOfficeEntry(position: Int($0.position), title: $0.title, weekendGross: Int($0.weekendGross), totalGross: Int($0.totalGross),
                                   weeksInRelease: Int($0.weeksInRelease), item: $0.item.map(MediaItem.init))
                }
            )
        }
    }
}

/// The latest chart for the person's region (Discover "For you").
@MainActor @Observable
final class BoxOfficeModel {
    enum Phase: Equatable { case idle, loading, loaded, failed }
    private let api: TipsarrAPI
    private(set) var chart: BoxOfficeChart?
    private(set) var phase: Phase = .idle

    init(api: TipsarrAPI) { self.api = api }

    func loadIfNeeded() async {
        if phase == .idle { await refresh() }
    }

    func refresh() async {
        if chart == nil { phase = .loading }
        do {
            chart = try await api.boxOffice()
            phase = .loaded
        } catch {
            if chart == nil { phase = .failed }
        }
    }
}

extension BoxOfficeChart {
    /// "Oct 2 – Oct 4 · US"
    var title: String {
        let parser = DateFormatter()
        parser.dateFormat = "yyyy-MM-dd"
        parser.locale = Locale(identifier: "en_US_POSIX")
        guard let from = parser.date(from: start), let to = parser.date(from: end) else { return region }
        return "\(from.formatted(.dateTime.month(.abbreviated).day())) – \(to.formatted(.dateTime.month(.abbreviated).day())) · \(region)"
    }

    /// Gross in the region's currency, compact ("$12M").
    func money(_ amount: Int) -> String {
        let currency = Locale(identifier: "und_\(region)").currency?.identifier ?? "USD"
        return amount.formatted(.currency(code: currency).notation(.compactName).precision(.fractionLength(0...1)))
    }
}

/// Ranked row on Discover: the chart's matched titles with the weekend gross, and a link to the full chart.
struct BoxOfficeRail: View {
    let model: BoxOfficeModel
    let api: TipsarrAPI

    var body: some View {
        Group {
            if let chart = model.chart {
                let entries = chart.entries.filter { $0.item != nil }
                if !entries.isEmpty {
                    VStack(alignment: .leading, spacing: Tokens.Spacing.md) {
                        HStack(alignment: .firstTextBaseline) {
                            VStack(alignment: .leading, spacing: 2) {
                                Text("nav.boxoffice").font(.title3.weight(.semibold)).accessibilityAddTraits(.isHeader)
                                Text(verbatim: chart.title).font(.footnote).foregroundStyle(Tokens.palette.mutedFg)
                            }
                            Spacer()
                            NavigationLink { BoxOfficeScreen(api: api, first: chart) } label: {
                                Text("m.boxoffice.full").font(.subheadline)
                            }
                            .foregroundStyle(Tokens.palette.mutedFg)
                        }
                        .padding(.horizontal, Tokens.Spacing.lg)
                        ScrollView(.horizontal, showsIndicators: false) {
                            LazyHStack(alignment: .top, spacing: Tokens.Spacing.md) {
                                ForEach(entries) { entry in
                                    if let item = entry.item {
                                        MediaLink(route: item.route, requestable: item.state == nil) {
                                            PosterCard(title: item.title, subtitle: chart.money(entry.weekendGross), posterPath: item.posterPath,
                                                       state: item.state, rank: entry.position)
                                                .frame(width: 120)
                                        }
                                        .buttonStyle(.plain)
                                    }
                                }
                            }
                            .padding(.horizontal, Tokens.Spacing.lg)
                        }
                    }
                }
            }
        }
        .task { await model.loadIfNeeded() }
    }
}

/// Every entry of the chart, with the region and the week to pick.
struct BoxOfficeScreen: View {
    let api: TipsarrAPI
    @State private var chart: BoxOfficeChart
    @State private var loading = false

    init(api: TipsarrAPI, first: BoxOfficeChart) {
        self.api = api
        _chart = State(initialValue: first)
    }

    var body: some View {
        List {
            Section {
                ForEach(chart.entries) { entry in row(entry) }
            } header: {
                Text(verbatim: chart.title).textCase(nil)
            }
            if chart.entries.isEmpty { Section { Text("m.boxoffice.empty").foregroundStyle(Tokens.palette.mutedFg) } }
        }
        .scrollContentBackground(.hidden)
        .readableColumn()
        .background(Tokens.palette.bg)
        .overlay { if loading { ProgressView() } }
        .navigationTitle("nav.boxoffice")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Menu {
                    Picker("m.boxoffice.region", selection: Binding(get: { chart.region }, set: { load(region: $0, week: nil) })) {
                        ForEach(chart.regions, id: \.self) { code in Text(verbatim: "\(SettingsView.flag(code)) \(code)").tag(code) }
                    }
                    Picker("m.boxoffice.week", selection: Binding(get: { chart.week }, set: { load(region: chart.region, week: $0) })) {
                        ForEach(chart.weeks, id: \.key) { week in Text(verbatim: week.label).tag(week.key) }
                    }
                } label: { Image(systemName: "slider.horizontal.3") }
                .accessibilityLabel(Text("library.filters"))
            }
        }
    }

    @ViewBuilder private func row(_ entry: BoxOfficeEntry) -> some View {
        let content = HStack(spacing: Tokens.Spacing.md) {
            Text(verbatim: String(entry.position)).font(.title3.weight(.bold)).monospacedDigit().frame(width: 28)
            RemoteImage(path: entry.item?.posterPath, size: .w185) {
                Image(systemName: "film").foregroundStyle(Tokens.palette.mutedFg)
            }
            .frame(width: 48, height: 72)
            .background(Tokens.palette.muted)
            .clipShape(.rect(cornerRadius: Tokens.Radius.sm))
            VStack(alignment: .leading, spacing: 2) {
                Text(verbatim: entry.item?.title ?? entry.title).font(.body.weight(.semibold)).lineLimit(2)
                Text(verbatim: chart.money(entry.weekendGross)).font(.footnote).foregroundStyle(Tokens.palette.mutedFg)
                Text(verbatim: "\(chart.money(entry.totalGross)) · \(L10n.string("m.boxoffice.weeks", String(entry.weeksInRelease)))")
                    .font(.caption).foregroundStyle(Tokens.palette.mutedFg)
            }
            Spacer(minLength: 0)
            if let state = entry.item?.state { StatusBadge(state: state, compact: true, solid: true) }
        }
        .frame(minHeight: Tokens.Size.touchTarget)
        if let item = entry.item {
            MediaLink(route: item.route, requestable: item.state == nil) { content }.buttonStyle(.plain)
        } else {
            content
        }
    }

    private func load(region: String, week: String?) {
        loading = true
        Task {
            defer { loading = false }
            if let next = try? await api.boxOffice(region: region, week: week) { chart = next }
        }
    }
}
