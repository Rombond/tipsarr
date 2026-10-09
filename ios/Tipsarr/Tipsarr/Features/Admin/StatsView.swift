import Charts
import SwiftUI

/// Watching statistics. Everyone sees their own; admins can pick another person or everyone.
struct StatsView: View {
    @Environment(\.appContext) private var context
    @State private var period: StatsPeriod = .all
    /// nil = me, "all" = everyone, else a user id.
    @State private var who: String?
    @State private var users: [Profile] = []
    @State private var report: StatsReport?
    @State private var error: APIError?
    @State private var loading = true

    private var isAdmin: Bool { context?.profile.isAdmin == true }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: Tokens.Spacing._2xl) {
                controls
                if let report { content(report) } else if let error {
                    ErrorState(error: error) { await load() }
                } else if loading {
                    ProgressView().frame(maxWidth: .infinity, minHeight: 200)
                }
            }
            .padding(Tokens.Spacing.lg)
            .frame(maxWidth: 720)
            .frame(maxWidth: .infinity)
        }
        .background(Tokens.palette.bg)
        .navigationTitle("stats.title")
        .navigationBarTitleDisplayMode(.inline)
        .task(id: StatsKey(period: period, who: who)) { await load() }
        .task { if isAdmin { users = (try? await context?.api.users()) ?? [] } }
        .refreshable { await load() }
    }

    private struct StatsKey: Equatable {
        var period: StatsPeriod
        var who: String?
    }

    // MARK: Controls

    private var controls: some View {
        VStack(alignment: .leading, spacing: Tokens.Spacing.md) {
            Picker("stats.period", selection: $period) {
                ForEach(StatsPeriod.allCases, id: \.self) { Text($0.title).tag($0) }
            }
            .pickerStyle(.segmented)
            if isAdmin, !users.isEmpty {
                Picker(selection: $who) {
                    Text("stats.me").tag(String?.none)
                    Text("stats.everyone").tag(String?.some("all"))
                    ForEach(users.filter { $0.id != context?.profile.id }, id: \.id) { user in
                        Text(verbatim: user.name).tag(String?.some(user.id))
                    }
                } label: {
                    Text("stats.user")
                }
                .pickerStyle(.menu)
            }
        }
    }

    // MARK: Content

    @ViewBuilder private func content(_ r: StatsReport) -> some View {
        if r.plays == 0 && r.titles == 0 {
            StateView(symbol: "chart.bar", title: "stats.empty").frame(minHeight: 280)
        } else {
            tiles(r)
            if !r.months.isEmpty { chartCard("stats.months") { monthsChart(r.months) } }
            if r.hoursOfDay.contains(where: { $0 > 0 }) { chartCard("stats.hours_of_day") { hoursChart(r.hoursOfDay) } }
            if r.weekdays.contains(where: { $0 > 0 }) { chartCard("stats.weekdays") { weekdaysChart(r.weekdays) } }
            if !r.genres.isEmpty { chartCard("stats.genres") { buckets(r.genres) } }
            if !r.decades.isEmpty { chartCard("stats.decades") { buckets(r.decades) } }
            if !r.topMovies.isEmpty { topList("stats.top_movies", r.topMovies) }
            if !r.topShows.isEmpty { topList("stats.top_shows", r.topShows) }
            Text(r.exact ? "stats.source_plugin" : "stats.source_estimate")
                .font(.footnote).foregroundStyle(Tokens.palette.mutedFg)
        }
    }

    private func tiles(_ r: StatsReport) -> some View {
        LazyVGrid(columns: [GridItem(.flexible(), spacing: Tokens.Spacing.md), GridItem(.flexible(), spacing: Tokens.Spacing.md)], spacing: Tokens.Spacing.md) {
            tile(r.exact ? "stats.tile_hours" : "stats.tile_hours_est", Self.hours(r.hours), nil)
            tile("stats.tile_plays", String(r.plays), nil)
            tile("stats.tile_titles", String(r.titles), L10n.string("stats.tile_titles_sub", String(r.movies), String(r.shows)))
            tile("stats.tile_requests", String(r.requestsMade), L10n.string("stats.tile_requests_sub", String(r.requestsAvailable), String(r.requestsDeclined)))
        }
    }

    private func tile(_ label: LText, _ value: String, _ sub: String?) -> some View {
        VStack(alignment: .leading, spacing: Tokens.Spacing.xs) {
            Text(label).font(.footnote).foregroundStyle(Tokens.palette.mutedFg)
            Text(verbatim: value).font(.title.weight(.bold))
            if let sub { Text(verbatim: sub).font(.caption).foregroundStyle(Tokens.palette.mutedFg) }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(Tokens.Spacing.md)
        .background(Tokens.palette.card, in: .rect(cornerRadius: Tokens.Radius.md))
        .overlay { RoundedRectangle(cornerRadius: Tokens.Radius.md).strokeBorder(Tokens.palette.border) }
        .accessibilityElement(children: .combine)
    }

    private func chartCard<Chart: View>(_ title: LText, @ViewBuilder _ chart: () -> Chart) -> some View {
        VStack(alignment: .leading, spacing: Tokens.Spacing.md) {
            Text(title).font(.headline).accessibilityAddTraits(.isHeader)
            chart()
        }
        .padding(Tokens.Spacing.md)
        .background(Tokens.palette.card, in: .rect(cornerRadius: Tokens.Radius.md))
        .overlay { RoundedRectangle(cornerRadius: Tokens.Radius.md).strokeBorder(Tokens.palette.border) }
    }

    // MARK: Charts

    private func monthsChart(_ months: [StatsMonth]) -> some View {
        Chart(months) { item in
            BarMark(x: .value("Month", Self.monthLabel(item.month)), y: .value("h", item.hours))
                .foregroundStyle(Tokens.palette.primary)
        }
        .chartYAxisLabel(L10n.string("stats.unit_h"))
        .frame(height: 180)
    }

    private func hoursChart(_ hours: [Double]) -> some View {
        Chart(Array(hours.enumerated()), id: \.offset) { index, value in
            BarMark(x: .value("Hour", index), y: .value("h", value)).foregroundStyle(Tokens.palette.primary)
        }
        .chartXAxis { AxisMarks(values: [0, 6, 12, 18, 23]) }
        .frame(height: 150)
    }

    private func weekdaysChart(_ days: [Double]) -> some View {
        // The server counts Monday first.
        let symbols = Calendar.current.shortStandaloneWeekdaySymbols
        let names = Array(symbols[1...]) + [symbols[0]]
        return Chart(Array(days.enumerated()), id: \.offset) { index, value in
            BarMark(x: .value("Day", names[min(index, 6)]), y: .value("h", value)).foregroundStyle(Tokens.palette.primary)
        }
        .frame(height: 150)
    }

    private func buckets(_ list: [StatsBucket]) -> some View {
        let top = Array(list.prefix(8))
        return Chart(top) { item in
            BarMark(x: .value("h", item.hours), y: .value("Name", item.name))
                .foregroundStyle(Tokens.palette.primary)
                .annotation(position: .trailing) {
                    Text(verbatim: Self.hours(item.hours)).font(.caption2).foregroundStyle(Tokens.palette.mutedFg)
                }
        }
        .chartYScale(domain: top.map(\.name))
        .frame(height: CGFloat(top.count) * 30 + 20)
    }

    // MARK: Top lists

    private func topList(_ title: LText, _ items: [StatsTop]) -> some View {
        VStack(alignment: .leading, spacing: Tokens.Spacing.md) {
            Text(title).font(.title3.weight(.bold)).accessibilityAddTraits(.isHeader)
            VStack(spacing: 0) {
                ForEach(Array(items.prefix(5).enumerated()), id: \.element.id) { index, item in
                    NavigationLink(value: item.route) {
                        HStack(spacing: Tokens.Spacing.md) {
                            Text(verbatim: String(index + 1)).font(.headline).foregroundStyle(Tokens.palette.mutedFg).frame(width: 20)
                            RemoteImage(serverPath: item.posterPath) { Image(systemName: "film").foregroundStyle(Tokens.palette.mutedFg) }
                                .frame(width: 40, height: 60)
                                .background(Tokens.palette.muted)
                                .clipShape(.rect(cornerRadius: 6))
                                .accessibilityHidden(true)
                            VStack(alignment: .leading, spacing: 2) {
                                Text(verbatim: item.title).font(.body.weight(.medium)).lineLimit(1).foregroundStyle(Tokens.palette.fg)
                                Text(verbatim: L10n.string("stats.n_plays", String(item.plays)) + " · " + Self.hours(item.hours))
                                    .font(.footnote).foregroundStyle(Tokens.palette.mutedFg)
                            }
                            Spacer(minLength: 0)
                        }
                        .padding(.vertical, Tokens.Spacing.sm)
                        .contentShape(.rect)
                    }
                    .buttonStyle(.plain)
                    Divider()
                }
            }
        }
    }

    // MARK: Data and formatting

    private func load() async {
        guard let context else { return }
        loading = true
        defer { loading = false }
        do {
            report = try await context.api.stats(period: period, user: who)
            error = nil
        } catch {
            if report == nil { self.error = APIError.from(error) }
        }
    }

    private static func hours(_ value: Double) -> String {
        value >= 10 ? "\(Int(value.rounded())) \(L10n.string("stats.unit_h"))" : "\(value.formatted(.number.precision(.fractionLength(1)))) \(L10n.string("stats.unit_h"))"
    }

    /// "2026-03" → "Mar".
    private static func monthLabel(_ month: String) -> String {
        guard let date = try? Date(month + "-01", strategy: .iso8601.year().month().day().dateSeparator(.dash)) else { return month }
        return date.formatted(.dateTime.month(.abbreviated))
    }
}
