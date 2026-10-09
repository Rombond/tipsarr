import Charts
import SwiftUI

/// Watching statistics, in the order of the web page: tiles, most watched (all, movies, shows), charts,
/// "asked for, never watched". Everyone sees their own; admins can pick someone or everyone.
struct StatsView: View {
    @Environment(\.appContext) private var context
    @State private var period: StatsPeriod = .all
    /// nil = me, "all" = everyone, else a user id.
    @State private var who: String?
    @State private var users: [Profile] = []
    @State private var report: StatsReport?
    @State private var unwatched: RequestPage?
    @State private var neverWatched: LibraryPage?
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
                unwatchedSection
                if isAdmin { neverWatchedSection }
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
            if let report {
                Text(report.exact ? "stats.source_plugin" : "stats.source_estimate")
                    .font(.footnote).foregroundStyle(Tokens.palette.mutedFg)
            }
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

    // MARK: Report

    @ViewBuilder private func content(_ r: StatsReport) -> some View {
        if isAdmin && r.pluginHint { pluginCard }
        if r.plays == 0 && r.titles == 0 {
            StateView(symbol: "chart.bar", title: "stats.empty").frame(minHeight: 240)
        } else {
            tiles(r)
            MostWatchedCarousel(title: "stats.top", titleKey: "stats.top", items: r.top)
            MostWatchedCarousel(title: "stats.top_movies", titleKey: "stats.top_movies", items: r.topMovies)
            MostWatchedCarousel(title: "stats.top_shows", titleKey: "stats.top_shows", items: r.topShows)
            if !r.genres.isEmpty { chartCard("stats.genres") { buckets(r.genres, byHours: true) } }
            if !r.decades.isEmpty { chartCard("stats.decades") { buckets(r.decades, byHours: false) } }
            if !r.months.isEmpty { chartCard("stats.months") { monthsChart(r.months) } }
            if r.weekdays.contains(where: { $0 > 0 }) { chartCard("stats.weekdays") { weekdaysChart(r.weekdays) } }
            if r.hoursOfDay.contains(where: { $0 > 0 }) { chartCard("stats.hours_of_day") { hoursChart(r.hoursOfDay) } }
        }
    }

    private var pluginCard: some View {
        VStack(alignment: .leading, spacing: Tokens.Spacing.sm) {
            Label { Text("stats.plugin_title").font(.headline) } icon: { Image(systemName: "info.circle") }
            Text("stats.plugin_text").font(.footnote).foregroundStyle(Tokens.palette.mutedFg)
            VStack(alignment: .leading, spacing: Tokens.Spacing.xs) {
                Text("stats.plugin_step1")
                Text("stats.plugin_step2")
                Text("stats.plugin_step3")
            }
            .font(.footnote)
        }
        .padding(Tokens.Spacing.md)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(Tokens.palette.muted, in: .rect(cornerRadius: Tokens.Radius.md))
    }

    private func tiles(_ r: StatsReport) -> some View {
        LazyVGrid(columns: [GridItem(.flexible(), spacing: Tokens.Spacing.md), GridItem(.flexible(), spacing: Tokens.Spacing.md)], spacing: Tokens.Spacing.md) {
            tile(r.exact ? "stats.tile_hours" : "stats.tile_hours_est", StatsFormat.hours(r.hours), L10n.string("stats.unit_hours_long"))
            tile("stats.tile_titles", String(r.titles), L10n.string("stats.tile_titles_sub", String(r.movies), String(r.shows)))
            tile("stats.tile_plays", String(r.plays), r.genres.first.map { L10n.string("stats.tile_plays_sub", $0.name) })
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
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(Tokens.palette.card, in: .rect(cornerRadius: Tokens.Radius.md))
        .overlay { RoundedRectangle(cornerRadius: Tokens.Radius.md).strokeBorder(Tokens.palette.border) }
    }

    // MARK: Charts

    private func buckets(_ list: [StatsBucket], byHours: Bool) -> some View {
        let top = Array(list.prefix(8))
        return Chart(top) { item in
            BarMark(x: .value("v", byHours && item.hours > 0 ? item.hours : Double(item.titles)), y: .value("Name", item.name))
                .foregroundStyle(Tokens.palette.primary)
                .annotation(position: .trailing) {
                    Text(verbatim: byHours && item.hours > 0 ? StatsFormat.hours(item.hours) : L10n.string("stats.n_titles", String(item.titles)))
                        .font(.caption2).foregroundStyle(Tokens.palette.mutedFg)
                }
        }
        .chartYScale(domain: top.map(\.name))
        .chartXAxis(.hidden)
        .frame(height: CGFloat(top.count) * 30 + 20)
    }

    private func monthsChart(_ months: [StatsMonth]) -> some View {
        Chart(months) { item in
            BarMark(x: .value("Month", Self.monthLabel(item.month)), y: .value("h", item.hours)).foregroundStyle(Tokens.palette.primary)
        }
        .frame(height: 180)
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

    private func hoursChart(_ hours: [Double]) -> some View {
        Chart(Array(hours.enumerated()), id: \.offset) { index, value in
            BarMark(x: .value("Hour", index), y: .value("h", value)).foregroundStyle(Tokens.palette.primary)
        }
        .chartXAxis { AxisMarks(values: [0, 3, 6, 9, 12, 15, 18, 21]) }
        .frame(height: 150)
    }

    // MARK: Asked for, never watched

    private var unwatchedSection: some View {
        VStack(alignment: .leading, spacing: Tokens.Spacing.md) {
            Text("stats.unwatched_title").font(.title3.weight(.bold)).accessibilityAddTraits(.isHeader)
            Text(who == nil || !isAdmin ? "stats.unwatched_mine" : "stats.unwatched_others").font(.footnote).foregroundStyle(Tokens.palette.mutedFg)
            if let unwatched {
                if unwatched.items.isEmpty {
                    Text("stats.unwatched_none").font(.subheadline).foregroundStyle(Tokens.palette.mutedFg)
                } else {
                    VStack(spacing: 0) {
                        ForEach(unwatched.items) { record in
                            NavigationLink(value: MediaRoute(type: record.type, tmdbId: record.tmdbId, title: record.title)) {
                                row(poster: record.posterPath, server: false, title: record.title,
                                    note: L10n.string("stats.unwatched_since", record.createdAt.formatted(date: .abbreviated, time: .omitted)))
                            }
                            .buttonStyle(.plain)
                            Divider()
                        }
                        if unwatched.total > unwatched.items.count {
                            Text(verbatim: L10n.string("stats.unwatched_more", String(unwatched.total - unwatched.items.count)))
                                .font(.footnote).foregroundStyle(Tokens.palette.mutedFg).padding(.top, Tokens.Spacing.sm)
                        }
                    }
                }
            }
        }
    }

    private var neverWatchedSection: some View {
        VStack(alignment: .leading, spacing: Tokens.Spacing.md) {
            Text("stats.never_title").font(.title3.weight(.bold)).accessibilityAddTraits(.isHeader)
            if let neverWatched {
                if neverWatched.items.isEmpty {
                    Text("stats.never_none").font(.subheadline).foregroundStyle(Tokens.palette.mutedFg)
                } else {
                    Text(verbatim: L10n.string("stats.never_hint", String(neverWatched.total))).font(.footnote).foregroundStyle(Tokens.palette.mutedFg)
                    VStack(spacing: 0) {
                        ForEach(neverWatched.items) { item in
                            NavigationLink(value: item.route) {
                                row(poster: item.posterPath, server: true, title: item.title, note: item.year.map(String.init) ?? "")
                            }
                            .buttonStyle(.plain)
                            Divider()
                        }
                    }
                }
            }
        }
    }

    private func row(poster: String?, server: Bool, title: String, note: String) -> some View {
        HStack(spacing: Tokens.Spacing.md) {
            Group {
                if server { RemoteImage(serverPath: poster) { Image(systemName: "film").foregroundStyle(Tokens.palette.mutedFg) } }
                else { RemoteImage(path: poster, size: .w92) { Image(systemName: "film").foregroundStyle(Tokens.palette.mutedFg) } }
            }
            .frame(width: 40, height: 60)
            .background(Tokens.palette.muted)
            .clipShape(.rect(cornerRadius: 6))
            .accessibilityHidden(true)
            VStack(alignment: .leading, spacing: 2) {
                Text(verbatim: title).font(.body.weight(.medium)).lineLimit(1).foregroundStyle(Tokens.palette.fg)
                if !note.isEmpty { Text(verbatim: note).font(.footnote).foregroundStyle(Tokens.palette.mutedFg) }
            }
            Spacer(minLength: 0)
        }
        .padding(.vertical, Tokens.Spacing.sm)
        .contentShape(.rect)
        .accessibilityElement(children: .combine)
    }

    // MARK: Data

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
        // Who the "never watched" list is about: the person, or everyone when an admin picked "everyone".
        let target = who == "all" ? nil : (who ?? context.profile.id)
        unwatched = try? await context.api.unwatchedRequests(user: target)
        if isAdmin, neverWatched == nil { neverWatched = try? await context.api.neverWatched() }
    }

    /// "2026-03" → "Mar".
    private static func monthLabel(_ month: String) -> String {
        guard let date = try? Date(month + "-01", strategy: .iso8601.year().month().day().dateSeparator(.dash)) else { return month }
        return date.formatted(.dateTime.month(.abbreviated))
    }
}
