import SwiftUI

/// Entry point used by every `NavigationLink(value: MediaRoute)`.
struct MediaDetailScreen: View {
    let route: MediaRoute
    @Environment(\.appContext) private var context

    var body: some View {
        if let context {
            MediaDetailView(route: route, context: context)
                .id(route)
        }
    }
}

struct MediaDetailView: View {
    @State private var model: MediaDetailModel
    @State private var sheet: Sheet?
    @State private var confirmCancel = false
    @Environment(ToastCenter.self) private var toast

    enum Sheet: Identifiable {
        case request, report
        var id: Self { self }
    }

    init(route: MediaRoute, context: AppContext) {
        _model = State(initialValue: MediaDetailModel(route: route, context: context))
    }

    var body: some View {
        Group {
            switch model.phase {
            case .loading:
                DetailSkeleton(title: model.route.title)
            case .failed(let error):
                ErrorState(error: error) { await model.load() }
                    .frame(maxHeight: .infinity)
            case .loaded:
                if let detail = model.detail { content(detail) }
            }
        }
        .background(Tokens.palette.bg)
        .navigationBarTitleDisplayMode(.inline)
        .toolbarBackgroundVisibility(.hidden, for: .navigationBar)
        .task { await model.load() }
        .sheet(item: $sheet) { item in
            switch item {
            case .request:
                RequestSheet(model: model) { record in
                    self.sheet = nil
                    announce(record)
                }
                .tipsarrSheet(detents: [.large])
            case .report:
                ReportIssueSheet(title: model.route.title) { kind, message in
                    try await model.report(kind: kind, message: message)
                    self.sheet = nil
                    toast.show(L10n.string("issue.toast_reported"))
                }
                .tipsarrSheet(detents: [.medium, .large])
            }
        }
        .confirmationDialog(
            Text(verbatim: L10n.string("m.request.cancel_confirm", model.route.title)),
            isPresented: $confirmCancel, titleVisibility: .visible
        ) {
            Button("m.request.cancel", role: .destructive) { Task { await run { try await model.cancelRequest() } } }
            Button("m.request.keep", role: .cancel) {}
        }
    }

    // MARK: Content

    private func content(_ detail: MediaDetail) -> some View {
        ScrollView {
            VStack(alignment: .leading, spacing: Tokens.Spacing.xl) {
                DetailHeader(detail: detail, badge: model.badge)
                    .padding(.bottom, -Tokens.Spacing.sm)
                VStack(alignment: .leading, spacing: Tokens.Spacing.xl) {
                    RatingsStrip(ratings: model.ratings)
                    actionRow(detail)
                    banners
                    overview(detail)
                    if detail.type == .tv, !model.regularSeasons.isEmpty { seasons }
                    facts(detail)
                    if !detail.cast.isEmpty { cast(detail) }
                    if !detail.recommendations.isEmpty { recommendations(detail) }
                    reportRow
                }
                .padding(.horizontal, Tokens.Spacing.lg)
                .frame(maxWidth: 720)
                .frame(maxWidth: .infinity)
            }
            .padding(.bottom, Tokens.Spacing._3xl)
        }
        .ignoresSafeArea(edges: .top)
        .refreshable { await model.load() }
    }

    // MARK: Actions

    private func actionRow(_ detail: MediaDetail) -> some View {
        VStack(alignment: .leading, spacing: Tokens.Spacing.md) {
            HStack(spacing: Tokens.Spacing.md) {
                mainButton
                menu(detail)
            }
            if case .inProgress(.downloading, _) = model.action, let request = model.request, let percent = request.progressPercent {
                ProgressBar(percent: percent)
                HStack {
                    Text(verbatim: L10n.string("req.stage.downloading_pct", percent))
                    if let eta = request.etaSeconds, eta > 0 {
                        Text(Duration.seconds(eta), format: .units(allowed: [.hours, .minutes], width: .abbreviated, maximumUnitCount: 2))
                    }
                }
                .font(.footnote)
                .foregroundStyle(Tokens.palette.mutedFg)
            }
        }
    }

    @ViewBuilder private var mainButton: some View {
        switch model.action {
        case .request:
            Button { Task { await startRequest() } } label: {
                Label { Text(choose(model.busy, "media.requesting", "media.request")) } icon: { Image(systemName: "plus") }
            }
            .buttonStyle(.tipsarr(.primary, fullWidth: true))
            .disabled(model.busy)
        case .inProgress(let state, let cancellable):
            Button { if cancellable { confirmCancel = true } } label: {
                Label { Text(state.title) } icon: { Image(systemName: state.symbol) }
            }
            .buttonStyle(.tipsarr(.secondary, fullWidth: true))
            .disabled(!cancellable)
        case .available(let url):
            if let url {
                Link(destination: url) {
                    Label { Text("m.detail.open_jellyfin") } icon: { Image(systemName: "play.fill") }
                }
                .buttonStyle(.tipsarr(.primary, fullWidth: true))
            } else {
                Label { Text(RequestState.available.title) } icon: { Image(systemName: RequestState.available.symbol) }
                    .font(.headline)
                    .foregroundStyle(RequestState.available.color)
                    .frame(maxWidth: .infinity, minHeight: Tokens.Size.touchTarget)
            }
        case .requestAgain:
            Button { Task { await startRequest() } } label: {
                Label { Text("m.detail.request_again") } icon: { Image(systemName: "arrow.counterclockwise") }
            }
            .buttonStyle(.tipsarr(.secondary, fullWidth: true))
            .disabled(model.busy)
        case .failed(let canRetry):
            if canRetry {
                Button { Task { await run { try await model.retryRequest() } } } label: {
                    Label { Text("common.retry") } icon: { Image(systemName: "arrow.clockwise") }
                }
                .buttonStyle(.tipsarr(.primary, fullWidth: true))
                .disabled(model.busy)
            } else {
                Label { Text(RequestState.failed.title) } icon: { Image(systemName: RequestState.failed.symbol) }
                    .font(.headline)
                    .foregroundStyle(RequestState.failed.color)
                    .frame(maxWidth: .infinity, minHeight: Tokens.Size.touchTarget)
            }
        }
    }

    private func menu(_ detail: MediaDetail) -> some View {
        Menu {
            Button {
                Task { await run { try await model.toggleWatchlist() } }
            } label: {
                Label { Text(choose(model.flags.watchlisted, "actions.on_watchlist", "actions.watchlist")) }
                    icon: { Image(systemName: model.flags.watchlisted ? "bookmark.fill" : "bookmark") }
            }
            Button {
                Task { await run { try await model.toggleHidden() } }
            } label: {
                Label { Text(choose(model.flags.blocklisted, "actions.show_again", "media.not_interested")) }
                    icon: { Image(systemName: model.flags.blocklisted ? "eye" : "eye.slash") }
            }
            if let key = detail.trailerKey, let url = URL(string: "https://www.youtube.com/watch?v=\(key)") {
                Link(destination: url) { Label { Text("actions.trailer") } icon: { Image(systemName: "play.rectangle") } }
            }
            Button { sheet = .report } label: {
                Label { Text("actions.report_short") } icon: { Image(systemName: "flag") }
            }
            if case .inProgress(_, true) = model.action {
                Button(role: .destructive) { confirmCancel = true } label: {
                    Label { Text("m.request.cancel") } icon: { Image(systemName: "xmark.circle") }
                }
            }
        } label: {
            Image(systemName: "ellipsis")
                .font(.headline)
                .frame(width: Tokens.Size.touchTarget, height: Tokens.Size.touchTarget)
                .background(Tokens.palette.muted, in: .circle)
                .foregroundStyle(Tokens.palette.fg)
        }
        .accessibilityLabel(Text("common.more_actions"))
    }

    @ViewBuilder private var banners: some View {
        switch model.action {
        case .requestAgain:
            let reason = model.request?.declineReason
            Banner(kind: .error, title: reason.map { LText.verbatim(L10n.string("req.reason", $0)) } ?? "m.detail.declined")
        case .failed:
            Banner(kind: .warning, title: "req.failed_generic")
        default:
            EmptyView()
        }
    }

    // MARK: Sections

    private func overview(_ detail: MediaDetail) -> some View {
        VStack(alignment: .leading, spacing: Tokens.Spacing.md) {
            Text("detail.overview").font(.title3.weight(.bold)).accessibilityAddTraits(.isHeader)
            if let tagline = detail.tagline {
                Text(verbatim: tagline).font(.subheadline.italic()).foregroundStyle(Tokens.palette.mutedFg)
            }
            if let overview = detail.overview {
                Text(verbatim: overview).font(.callout).lineSpacing(3)
            } else {
                Text("m.detail.no_overview").font(.callout).foregroundStyle(Tokens.palette.mutedFg)
            }
            if !detail.genres.isEmpty {
                FlowLayout(spacing: Tokens.Spacing.sm) {
                    ForEach(detail.genres, id: \.self) { genre in
                        Text(verbatim: genre)
                            .font(.footnote.weight(.medium))
                            .padding(.horizontal, Tokens.Spacing.md)
                            .padding(.vertical, Tokens.Spacing.xs + 2)
                            .background(Tokens.palette.muted, in: .capsule)
                    }
                }
            }
        }
    }

    private var seasons: some View {
        VStack(alignment: .leading, spacing: Tokens.Spacing.md) {
            Text("seasons.title").font(.title3.weight(.bold)).accessibilityAddTraits(.isHeader)
            ForEach(model.regularSeasons) { season in
                HStack(spacing: Tokens.Spacing.md) {
                    VStack(alignment: .leading, spacing: 2) {
                        Text(verbatim: season.name).font(.body.weight(.semibold))
                        Text(verbatim: seasonMeta(season)).font(.footnote).foregroundStyle(Tokens.palette.mutedFg)
                    }
                    Spacer()
                    if model.coveredSeasons.contains(season.number) {
                        StatusBadge(state: model.request?.state ?? .requested)
                    }
                }
                .padding(Tokens.Spacing.md)
                .background(Tokens.palette.card, in: .rect(cornerRadius: Tokens.Radius.md))
                .overlay { RoundedRectangle(cornerRadius: Tokens.Radius.md).strokeBorder(Tokens.palette.border) }
                .accessibilityElement(children: .combine)
            }
        }
    }

    private func seasonMeta(_ season: SeasonInfo) -> String {
        let episodes = Plural.text("seasons.episodes", count: season.episodeCount)
        guard let year = season.airDate.flatMap({ $0.count >= 4 ? String($0.prefix(4)) : nil }) else { return episodes }
        return "\(episodes) · \(year)"
    }

    private func facts(_ detail: MediaDetail) -> some View {
        VStack(alignment: .leading, spacing: Tokens.Spacing.md) {
            Text("detail.title_fallback").font(.title3.weight(.bold)).accessibilityAddTraits(.isHeader)
            VStack(spacing: 0) {
                if let status = detail.status, !status.isEmpty {
                    let key = "status.\(status)"
                    let text = AppLanguage.bundle.localizedString(forKey: key, value: status, table: nil)
                    FactRow(label: "fact.status", value: text)
                }
                if let date = detail.releaseDate.flatMap(Self.parseDate) {
                    FactRow(label: "fact.release_date", value: date.formatted(date: .long, time: .omitted))
                }
                if let minutes = detail.runtimeMinutes, minutes > 0 {
                    FactRow(label: "fact.runtime", value: Duration.seconds(minutes * 60).formatted(.units(allowed: [.hours, .minutes], width: .narrow)))
                }
                if let language = detail.originalLanguage, let name = Locale.current.localizedString(forLanguageCode: language) {
                    FactRow(label: "fact.language", value: name.localizedCapitalized)
                }
                if !detail.studios.isEmpty {
                    FactRow(label: detail.type == .tv ? "fact.network" : "fact.studio", value: detail.studios.prefix(3).formatted())
                }
            }
        }
    }

    private static func parseDate(_ text: String) -> Date? {
        try? Date(text, strategy: .iso8601.year().month().day())
    }

    private func cast(_ detail: MediaDetail) -> some View {
        VStack(alignment: .leading, spacing: Tokens.Spacing.md) {
            Text("detail.cast").font(.title3.weight(.bold)).accessibilityAddTraits(.isHeader)
            ScrollView(.horizontal, showsIndicators: false) {
                LazyHStack(alignment: .top, spacing: Tokens.Spacing.md) {
                    ForEach(detail.cast.prefix(15)) { member in
                        NavigationLink(value: PersonRoute(id: member.id, name: member.name)) {
                        VStack(spacing: Tokens.Spacing.xs) {
                            RemoteImage(path: member.profilePath, size: .w185) {
                                Image(systemName: "person.fill").foregroundStyle(Tokens.palette.mutedFg)
                            }
                            .frame(width: 64, height: 64)
                            .background(Tokens.palette.muted)
                            .clipShape(.circle)
                            Text(verbatim: member.name).font(.caption.weight(.medium)).lineLimit(1)
                            if let character = member.character, !character.isEmpty {
                                Text(verbatim: character).font(.caption2).foregroundStyle(Tokens.palette.mutedFg).lineLimit(1)
                            }
                        }
                        .frame(width: 80)
                        .accessibilityElement(children: .combine)
                        }
                        .buttonStyle(.plain)
                    }
                }
            }
            .scrollClipDisabled()
        }
    }

    private func recommendations(_ detail: MediaDetail) -> some View {
        VStack(alignment: .leading, spacing: Tokens.Spacing.md) {
            Text("detail.more_like").font(.title3.weight(.bold)).accessibilityAddTraits(.isHeader)
            ScrollView(.horizontal, showsIndicators: false) {
                LazyHStack(alignment: .top, spacing: Tokens.Spacing.md) {
                    ForEach(detail.recommendations.prefix(15)) { item in
                        NavigationLink(value: item.route) {
                            PosterCard(item: item)
                                .frame(width: 110)
                        }
                        .buttonStyle(.plain)
                    }
                }
            }
            .scrollClipDisabled()
        }
    }

    private var reportRow: some View {
        Button { sheet = .report } label: {
            HStack {
                Label { Text("actions.report_short") } icon: { Image(systemName: "flag") }
                Spacer()
                Image(systemName: "chevron.right").foregroundStyle(Tokens.palette.mutedFg)
            }
            .font(.body.weight(.medium))
            .padding(.horizontal, Tokens.Spacing.lg)
            .frame(minHeight: Tokens.Size.touchTarget + Tokens.Spacing.sm)
            .background(Tokens.palette.card, in: .rect(cornerRadius: Tokens.Radius.md))
            .overlay { RoundedRectangle(cornerRadius: Tokens.Radius.md).strokeBorder(Tokens.palette.border) }
        }
        .buttonStyle(.plain)
    }

    // MARK: Flow

    private func startRequest() async {
        if await model.prepareRequest() {
            sheet = .request
            return
        }
        do {
            let seasons = model.route.type == .tv ? model.regularSeasons.map(\.number) : nil
            announce(try await model.submitRequest(seasons: seasons, profileID: nil, folder: nil))
        } catch {
            toast.show(APIError.from(error).localizedMessage, kind: .error)
        }
    }

    private func announce(_ record: RequestRecord) {
        let title = model.route.title
        let key = record.state == .requested ? "media.toast_requested" : (record.dryRun ? "media.toast_approved_dry" : "media.toast_approved")
        toast.show(L10n.string(key, title))
    }

    private func run(_ work: () async throws -> Void) async {
        do { try await work() } catch { toast.show(APIError.from(error).localizedMessage, kind: .error) }
    }
}

private struct FactRow: View {
    let label: LText
    let value: String

    var body: some View {
        HStack(alignment: .firstTextBaseline) {
            Text(label).foregroundStyle(Tokens.palette.mutedFg)
            Spacer(minLength: Tokens.Spacing.lg)
            Text(verbatim: value).multilineTextAlignment(.trailing)
        }
        .font(.subheadline)
        .padding(.vertical, Tokens.Spacing.sm)
        .overlay(alignment: .bottom) { Divider() }
        .accessibilityElement(children: .combine)
    }
}
