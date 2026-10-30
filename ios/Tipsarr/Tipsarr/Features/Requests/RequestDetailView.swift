import SwiftUI

struct RequestDetailScreen: View {
    let record: RequestRecord
    let model: RequestsModel

    var body: some View {
        RequestDetailView(initial: record, model: model).id(record.id)
    }
}

struct RequestDetailView: View {
    let model: RequestsModel
    @State private var record: RequestRecord
    @State private var declining = false
    @State private var confirmDelete = false
    @State private var editing = false
    @State private var options: RequestOptions?
    @Environment(\.appContext) private var context
    @Environment(ToastCenter.self) private var toast
    @Environment(\.dismiss) private var dismiss
    @Environment(\.liveUpdates) private var live

    init(initial: RequestRecord, model: RequestsModel) {
        _record = State(initialValue: initial)
        self.model = model
    }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: Tokens.Spacing.xl) {
                header
                if record.state == .downloading, let percent = live?.progress[record.id]?.percent ?? record.progressPercent { progress(percent) }
                banner
                if model.canViewAll, let name = record.requestedBy, !name.isEmpty { requester(name) }
                VStack(alignment: .leading, spacing: Tokens.Spacing.md) {
                    Text("m.requests.progress").font(.title3.weight(.bold)).accessibilityAddTraits(.isHeader)
                    Timeline(steps: steps)
                }
                facts
                actions
            }
            .padding(Tokens.Spacing.lg)
            .frame(maxWidth: 720)
            .frame(maxWidth: .infinity)
        }
        .background(Tokens.palette.bg)
        .navigationTitle("requests.title")
        .navigationBarTitleDisplayMode(.inline)
        .refreshable { await reload() }
        .task { await reload() }
        .task { if model.canManage { options = try? await context?.api.requestOptions(record.type) } }
        .sheet(isPresented: $editing) {
            RequestOptionsSheet(record: record) { updated in
                record = updated
                editing = false
                toast.show(L10n.string("common.saved"))
            }
            .tipsarrSheet(detents: [.medium])
        }
        .onChange(of: live?.requestsTick) { Task { await reload() } }
        .sheet(isPresented: $declining) {
            DeclineSheet(title: record.title) { reason in
                record = try await model.decline(record, reason: reason)
                declining = false
            }
            .tipsarrSheet(detents: [.medium])
        }
    }

    // MARK: Sections

    private var header: some View {
        HStack(alignment: .top, spacing: Tokens.Spacing.lg) {
            MediaLink(route: MediaRoute(type: record.type, tmdbId: record.tmdbId, title: record.title)) {
                RemoteImage(path: record.posterPath, size: .w342) {
                    Image(systemName: "film").font(.title).foregroundStyle(Tokens.palette.mutedFg)
                }
                .frame(width: 96, height: 96 * Tokens.Size.posterRatio)
                .background(Tokens.palette.muted)
                .clipShape(.rect(cornerRadius: Tokens.Radius.md))
            }
            .buttonStyle(.plain)
            .accessibilityLabel(Text("m.requests.open_title"))
            VStack(alignment: .leading, spacing: Tokens.Spacing.sm) {
                Text(verbatim: record.title).font(.title3.weight(.bold)).accessibilityAddTraits(.isHeader)
                Text(choose(record.type == .tv, "type.tv", "type.movie")).font(.footnote).foregroundStyle(Tokens.palette.mutedFg)
                StatusBadge(state: record.state)
            }
        }
    }

    private func progress(_ percent: Int) -> some View {
        VStack(alignment: .leading, spacing: Tokens.Spacing.sm) {
            ProgressBar(percent: percent)
            HStack {
                Text(verbatim: L10n.string("req.stage.downloading_pct", String(percent)))
                if let eta = live?.progress[record.id]?.etaSeconds ?? record.etaSeconds, eta > 0 {
                    Text(Duration.seconds(eta), format: .units(allowed: [.hours, .minutes], width: .abbreviated, maximumUnitCount: 2))
                }
            }
            .font(.footnote)
            .foregroundStyle(Tokens.palette.mutedFg)
        }
    }

    @ViewBuilder private var banner: some View {
        if record.state == .declined {
            Banner(kind: .error, title: record.declineReason.map { LText.verbatim(L10n.string("req.reason", $0)) } ?? "m.detail.declined")
        } else if record.state == .failed {
            Banner(kind: .warning, title: "req.failed_generic",
                   message: model.canManage ? record.error.map { LText.verbatim($0) } : nil)
        }
    }

    private func requester(_ name: String) -> some View {
        VStack(alignment: .leading, spacing: Tokens.Spacing.sm) {
            Text("m.requests.requested_by").font(.footnote.weight(.semibold)).foregroundStyle(Tokens.palette.mutedFg)
            HStack(spacing: Tokens.Spacing.md) {
                Text(verbatim: String(name.prefix(1)).uppercased())
                    .font(.headline)
                    .frame(width: 36, height: 36)
                    .background(Tokens.palette.muted, in: .circle)
                VStack(alignment: .leading) {
                    Text(verbatim: name).font(.body.weight(.semibold))
                    Text(record.createdAt, format: .relative(presentation: .named)).font(.footnote).foregroundStyle(Tokens.palette.mutedFg)
                }
            }
            .accessibilityElement(children: .combine)
        }
    }

    /// Who asked, the chosen quality and folder: people who manage requests only (others see their own request without these).
    @ViewBuilder private var facts: some View {
        if model.canManage {
            VStack(alignment: .leading, spacing: Tokens.Spacing.md) {
                Text("m.requests.details").font(.title3.weight(.bold)).accessibilityAddTraits(.isHeader)
                VStack(spacing: 0) {
                    if let name = record.requestedBy, !name.isEmpty {
                        let isMe = name == context?.profile.name
                        DetailFact(label: "m.requests.requested_by", value: isMe ? L10n.string("common.you").trimmingCharacters(in: CharacterSet(charactersIn: "()")).localizedCapitalized : name)
                    }
                    if let name = record.decidedBy, !name.isEmpty { DetailFact(label: "m.requests.decided_by", value: name) }
                    if !record.seasons.isEmpty {
                        DetailFact(label: "fact.seasons", value: record.seasons.sorted().map(String.init).formatted())
                    }
                    DetailFact(label: "m.requests.quality_profile", value: profileName)
                    DetailFact(label: "m.requests.root_folder", value: record.rootFolder ?? options?.defaultFolder ?? L10n.string("m.requests.default_value"))
                    if record.dryRun { DetailFact(label: "m.requests.dry_run", value: "✓") }
                    if record.imported { DetailFact(label: "m.requests.imported", value: "✓") }
                }
            }
        }
        if !record.seasons.isEmpty { seasonProgress }
    }

    private var profileName: String {
        let id = record.qualityProfileId ?? options?.defaultProfileID
        return options?.profiles.first { $0.id == id }?.name ?? id.map(String.init) ?? L10n.string("m.requests.default_value")
    }

    /// One bar per requested season: how much of it Sonarr has or is downloading (as on the web).
    private var seasonProgress: some View {
        VStack(alignment: .leading, spacing: Tokens.Spacing.sm) {
            ForEach(record.seasons.sorted(), id: \.self) { number in
                let percent = record.seasonProgress[number]
                HStack(spacing: Tokens.Spacing.md) {
                    Text(verbatim: L10n.string("req.season_badge", String(number))).font(.subheadline.weight(.semibold))
                    ProgressBar(percent: percent ?? 0, tint: percent == 100 ? Tokens.Status.available : Tokens.Status.downloading)
                    Text(verbatim: percent.map { "\($0)%" } ?? "–")
                        .font(.footnote.monospacedDigit())
                        .foregroundStyle(Tokens.palette.mutedFg)
                        .frame(minWidth: 44, alignment: .trailing)
                }
                .accessibilityElement(children: .combine)
            }
        }
        .padding(.top, Tokens.Spacing.sm)
    }

    @ViewBuilder private var actions: some View {
        VStack(spacing: Tokens.Spacing.sm) {
            if model.canManage && record.state == .requested {
                HStack(spacing: Tokens.Spacing.md) {
                    Button { Task { await run { record = try await model.approve(record) } } } label: {
                        Label { Text("req.approve") } icon: { Image(systemName: "checkmark") }
                    }
                    .buttonStyle(.tipsarr(.primary, fullWidth: true))
                    Button { declining = true } label: {
                        Label { Text("req.decline") } icon: { Image(systemName: "xmark") }
                    }
                    .buttonStyle(.tipsarr(.secondary, fullWidth: true))
                }
            }
            if model.canManage && record.state == .failed {
                Button { Task { await run { record = try await model.retry(record) } } } label: {
                    Label { Text("req.retry") } icon: { Image(systemName: "arrow.clockwise") }
                }
                .buttonStyle(.tipsarr(.primary, fullWidth: true))
            }
            // Same rule as the web: the requester or an admin may change the quality profile until it is sent.
            if !record.imported, record.state == .requested || record.state == .failed {
                Button { editing = true } label: {
                    Label { Text("req.change_options_short") } icon: { Image(systemName: "slider.horizontal.3") }
                }
                .buttonStyle(.tipsarr(.secondary, fullWidth: true))
            }
            if model.canDelete(record) {
                Button { confirmDelete = true } label: { Text("m.request.cancel") }
                    .buttonStyle(.tipsarr(.ghost, fullWidth: true))
                    .foregroundStyle(Tokens.palette.destructive)
                    .confirmationDialog(Text(verbatim: L10n.string("m.request.cancel_confirm", record.title)), isPresented: $confirmDelete, titleVisibility: .visible) {
                        Button("m.request.cancel", role: .destructive) {
                            Task {
                                await run {
                                    try await model.delete(record)
                                    dismiss()
                                }
                            }
                        }
                        Button("m.request.keep", role: .cancel) {}
                    }
            }
        }
        .disabled(model.busy.contains(record.id))
    }

    // MARK: Timeline

    private var steps: [Timeline.Step] {
        switch record.state {
        case .declined:
            return [.init(title: RequestState.requested.title, detail: detail(.requested), status: .done),
                    .init(title: RequestState.declined.title, detail: detail(.declined), status: .failed)]
        case .failed:
            return [.init(title: RequestState.requested.title, detail: detail(.requested), status: .done),
                    .init(title: RequestState.approved.title, detail: detail(.approved), status: .done),
                    .init(title: RequestState.failed.title, detail: detail(.failed), status: .failed)]
        default:
            let order: [RequestState] = [.requested, .approved, .searching, .downloading, .available]
            let current = order.firstIndex(of: record.state == .partial ? .available : record.state) ?? 0
            return order.enumerated().map { index, state in
                let percent = state == .downloading ? record.progressPercent.map { "\($0)%" } : nil
                let status: Timeline.Step.Status = record.state == .available || index < current ? .done : (index == current ? .current : .todo)
                return .init(title: state.title, detail: index <= current ? detail(state, extra: percent) : "", status: status)
            }
        }
    }

    /// When the step happened (latest time, from the request history), who did it, then the decline reason or the error
    /// on a second line. A server older than 0.11 sends no history: only the request date and the approver show then.
    private func detail(_ state: RequestState, extra: String? = nil) -> String {
        let event = record.event(state)
        let at = event?.at ?? (state == .requested ? record.createdAt : nil)
        let date = at.map { (record.approximateHistory && state != .requested ? "≈ " : "") + $0.formatted(.dateTime.month(.abbreviated).day().hour().minute()) }
        let name = state == .requested ? nil : event?.by ?? (state == .approved || state == .declined ? record.decidedBy : nil)
        let who = name.flatMap { $0.isEmpty ? nil : L10n.string("m.request.approved_by", $0) }
        let line = [date, who, extra].compactMap { $0 }.joined(separator: " · ")
        return [line, event?.note ?? ""].filter { !$0.isEmpty }.joined(separator: "\n")
    }

    // MARK: Data

    private func reload() async {
        guard let context, let fresh = try? await context.api.request(id: record.id) else { return }
        record = fresh
    }

    private func run(_ work: () async throws -> Void) async {
        do { try await work() } catch { toast.show(APIError.from(error).localizedMessage, kind: .error) }
    }
}

private struct DetailFact: View {
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

/// Vertical steps: done (green check), current, todo, failed.
struct Timeline: View {
    struct Step: Identifiable {
        enum Status { case done, current, todo, failed }
        let id = UUID()
        var title: LText
        var detail: String
        var status: Status
    }

    let steps: [Step]

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            ForEach(Array(steps.enumerated()), id: \.element.id) { index, step in
                HStack(alignment: .top, spacing: Tokens.Spacing.md) {
                    VStack(spacing: 0) {
                        marker(step.status)
                        if index < steps.count - 1 {
                            Rectangle()
                                .fill(step.status == .done ? Tokens.Status.available : Tokens.palette.border)
                                .frame(width: 2, height: 32)
                        }
                    }
                    VStack(alignment: .leading, spacing: 2) {
                        Text(step.title).font(.body.weight(step.status == .todo ? .medium : .bold))
                            .foregroundStyle(step.status == .todo ? Tokens.palette.mutedFg : Tokens.palette.fg)
                        if !step.detail.isEmpty {
                            Text(verbatim: step.detail).font(.footnote).foregroundStyle(Tokens.palette.mutedFg)
                        }
                    }
                    .padding(.top, 2)
                    Spacer(minLength: 0)
                }
                .accessibilityElement(children: .combine)
            }
        }
    }

    @ViewBuilder private func marker(_ status: Step.Status) -> some View {
        switch status {
        case .done:
            Image(systemName: "checkmark.circle.fill").font(.title2).foregroundStyle(Tokens.Status.available)
        case .failed:
            Image(systemName: "xmark.circle.fill").font(.title2).foregroundStyle(Tokens.Status.failed)
        case .current:
            Image(systemName: "circle.dotted.circle").font(.title2).foregroundStyle(Tokens.Status.downloading)
        case .todo:
            Image(systemName: "circle").font(.title2).foregroundStyle(Tokens.palette.border)
        }
    }
}

/// Quality profile (and folder) of a pending or failed request, as the web dialog.
private struct RequestOptionsSheet: View {
    let record: RequestRecord
    var onSaved: (RequestRecord) -> Void

    @Environment(\.appContext) private var context
    @Environment(\.dismiss) private var dismiss
    @State private var options: RequestOptions?
    @State private var profileID: Int?
    @State private var folder: String?
    @State private var busy = false
    @State private var errorText: String?

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: Tokens.Spacing.xl) {
                Text(verbatim: L10n.string("req.change_options", record.title)).font(.title3.weight(.bold))
                if let options {
                    VStack(spacing: Tokens.Spacing.sm) {
                        OptionRow(symbol: "slider.horizontal.3", title: "req.quality_profile") {
                            Menu {
                                ForEach(options.profiles) { profile in
                                    Button {
                                        profileID = profile.id
                                    } label: {
                                        if profileID == profile.id { Image(systemName: "checkmark") }
                                        Text(verbatim: profile.name)
                                    }
                                }
                            } label: { menuLabel(options.profiles.first { $0.id == profileID }?.name ?? "") }
                        }
                        if !options.rootFolders.isEmpty {
                            OptionRow(symbol: "externaldrive", title: "req.root_folder") {
                                Menu {
                                    ForEach(options.rootFolders) { root in
                                        Button {
                                            folder = root.path
                                        } label: {
                                            if folder == root.path { Image(systemName: "checkmark") }
                                            Text(verbatim: root.path)
                                        }
                                    }
                                } label: { menuLabel(folder ?? options.defaultFolder) }
                            }
                        }
                    }
                } else if errorText == nil {
                    ProgressView().frame(maxWidth: .infinity)
                }
                if let errorText { Banner(kind: .error, title: LText.verbatim(errorText)) }
                VStack(spacing: Tokens.Spacing.sm) {
                    Button { Task { await save() } } label: { Text(busy ? "common.saving" : "common.save") }
                        .buttonStyle(.tipsarr(.primary, fullWidth: true))
                        .disabled(busy || options == nil)
                    Button("common.cancel") { dismiss() }
                        .buttonStyle(.tipsarr(.tonal, fullWidth: true))
                }
            }
            .padding(Tokens.Spacing.xl)
        }
        .task { await load() }
    }

    private func menuLabel(_ text: String) -> some View {
        HStack(spacing: Tokens.Spacing.xs) {
            Text(verbatim: text).lineLimit(1).truncationMode(.middle)
            Image(systemName: "chevron.up.chevron.down").font(.caption2)
        }
        .foregroundStyle(Tokens.palette.mutedFg)
    }

    private func load() async {
        do {
            guard let o = try await context?.api.requestOptions(record.type) else { dismiss(); return }
            options = o
            profileID = (record.qualityProfileId ?? 0) != 0 ? record.qualityProfileId : o.defaultProfileID
            folder = record.rootFolder ?? o.defaultFolder
        } catch {
            errorText = APIError.from(error).localizedMessage
        }
    }

    /// Only what changed from the instance defaults is sent, like the web.
    private func save() async {
        guard let options, let api = context?.api else { return }
        busy = true
        errorText = nil
        defer { busy = false }
        do {
            let updated = try await api.updateRequestOptions(
                id: record.id,
                profileID: profileID != options.defaultProfileID ? profileID : nil,
                folder: !options.rootFolders.isEmpty && folder != options.defaultFolder ? folder : nil
            )
            onSaved(updated)
        } catch {
            errorText = APIError.from(error).localizedMessage
        }
    }
}
