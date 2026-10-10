import SwiftUI

struct RequestsView: View {
    @State private var model: RequestsModel
    @State private var filter: RequestFilter = .all
    @State private var declining: RequestRecord?
    @State private var deleting: RequestRecord?
    @Environment(ToastCenter.self) private var toast
    @Environment(\.liveUpdates) private var live
    @Environment(\.detailPane) private var pane
    @Binding var path: NavigationPath
    /// Selected request id in the two-column layout.
    @Binding var selection: String?
    @State private var width: CGFloat = 0
    var openDiscover: () -> Void = {}

    init(api: TipsarrAPI, isAdmin: Bool, path: Binding<NavigationPath>, selection: Binding<String?>, openDiscover: @escaping () -> Void = {}) {
        _model = State(initialValue: RequestsModel(api: api, isAdmin: isAdmin))
        _path = path
        _selection = selection
        self.openDiscover = openDiscover
    }

    /// Two columns (list and detail) when the screen is wide enough, a stack otherwise.
    private var split: Bool { width >= 700 }

    var body: some View {
        Group {
            if split { splitBody } else { stackBody }
        }
        .onGeometryChange(for: CGFloat.self, of: { $0.size.width }) { width = $0 }
        .onAppear { pane?.requestsModel = model }
        .task(id: filter) { await model.load(filter) }
        .onChange(of: live?.requestsTick) { Task { await model.load(filter) } }
        .sheet(item: $declining) { record in
            DeclineSheet(title: record.title) { reason in
                try await model.decline(record, reason: reason)
                declining = nil
            }
            .tipsarrSheet(detents: [.medium])
        }
        .confirmationDialog(
            Text(verbatim: L10n.string("m.request.cancel_confirm", deleting?.title ?? "")),
            isPresented: Binding(get: { deleting != nil }, set: { if !$0 { deleting = nil } }),
            titleVisibility: .visible, presenting: deleting
        ) { record in
            Button("m.request.cancel", role: .destructive) {
                Task { await run { try await model.delete(record) } }
            }
            Button("m.request.keep", role: .cancel) {}
        }
    }

    private var stackBody: some View {
        NavigationStack(path: $path) {
            list(selecting: nil)
                .paneInset()
                .background(Tokens.palette.bg)
                .navigationTitle("requests.title")
                .navigationDestination(for: RequestRecord.self) { record in
                    RequestDetailScreen(record: record, model: model).paneInset()
                }
                .mediaDestinations()
        }
    }

    private var selectedRecord: RequestRecord? { model.items.first { $0.id == selection } }

    private var splitBody: some View {
        NavigationSplitView(columnVisibility: .constant(.all)) {
            list(selecting: $selection)
                .background(Tokens.palette.bg)
                .navigationTitle("requests.title")
                .navigationSplitViewColumnWidth(min: 340, ideal: 400, max: 460)
        } detail: {
            NavigationStack {
                if let record = selectedRecord {
                    RequestDetailView(initial: record, model: model).id(record.id)
                } else {
                    StateView(symbol: "checklist", title: "m.requests.select").background(Tokens.palette.bg)
                }
            }
            .mediaDestinations()
        }
        .navigationSplitViewStyle(.balanced)
        // The first request is shown as soon as the list arrives.
        .onChange(of: model.items.first?.id) { _, first in
            if selection == nil || !model.items.contains(where: { $0.id == selection }) { selection = first }
        }
    }

    /// First row of the list, so it scrolls with the content and nothing is laid over it.
    private var filters: some View {
        ScrollView(.horizontal, showsIndicators: false) {
            HStack(spacing: Tokens.Spacing.sm) {
                ForEach(RequestFilter.allCases, id: \.self) { item in
                    Chip(title: item.title, isSelected: filter == item, count: item == .all ? nil : model.counts.count(item)) { filter = item }
                }
            }
            .padding(.horizontal, Tokens.Spacing.lg)
        }
        .listRowInsets(.init())
        .listRowBackground(Color.clear)
        .listRowSeparator(.hidden)
    }

    private func stateRow<S: View>(@ViewBuilder _ state: () -> S) -> some View {
        state()
            .frame(minHeight: 360)
            .listRowInsets(.init())
            .listRowBackground(Color.clear)
            .listRowSeparator(.hidden)
    }

    private func list(selecting: Binding<String?>?) -> some View {
        List(selection: selecting ?? .constant(nil)) {
            filters
            switch model.phase {
            case .idle, .loading:
                ForEach(0..<6, id: \.self) { _ in
                    HStack(alignment: .top, spacing: Tokens.Spacing.md) {
                        RoundedRectangle(cornerRadius: Tokens.Radius.sm).fill(Tokens.palette.border).frame(width: 64, height: 96)
                        VStack(alignment: .leading, spacing: Tokens.Spacing.sm) {
                            SkeletonBlock(height: 16).frame(width: 180)
                            SkeletonBlock(height: 12).frame(width: 110)
                            SkeletonBlock(height: 12).frame(width: 140)
                        }
                    }
                    .skeleton()
                    .listRowBackground(Color.clear)
                }
            case .failed(let error):
                stateRow { ErrorState(error: error) { await model.load(filter) } }
            case .loaded where model.items.isEmpty:
                stateRow {
                    if filter == .all {
                        StateView(symbol: "checklist", title: "requests.empty_all", message: "requests.empty_hint",
                                  actionTitle: "m.requests.empty_cta", action: openDiscover)
                    } else {
                        StateView(symbol: "checklist",
                                  title: LText.verbatim(L10n.string("requests.empty_tab", filter.title.resolved)))
                    }
                }
            case .loaded:
                ForEach(model.items) { record in
                    RequestRow(record: record, showRequester: model.isAdmin, busy: model.busy.contains(record.id),
                               progress: live?.progress[record.id], link: selecting == nil,
                               onApprove: { Task { await approve(record) } },
                               onDecline: { declining = record })
                        .listRowInsets(.init(top: Tokens.Spacing.md, leading: Tokens.Spacing.lg, bottom: Tokens.Spacing.md, trailing: Tokens.Spacing.lg))
                        .listRowBackground(Color.clear)
                        .swipeActions(edge: .trailing, allowsFullSwipe: false) {
                            if model.canDelete(record) {
                                Button(role: .destructive) { deleting = record } label: {
                                    Label("m.requests.cancel_swipe", systemImage: "trash")
                                }
                            }
                        }
                        .tag(record.id)
                        .task { await model.loadMore(after: record) }
                }
                if model.loadingMore { ProgressView().frame(maxWidth: .infinity).listRowBackground(Color.clear) }
            }
        }
        .listStyle(.plain)
        .refreshable { await model.load(filter) }
    }

    private func approve(_ record: RequestRecord) async {
        await run {
            let updated = try await model.approve(record)
            toast.show(L10n.string(updated.dryRun ? "media.toast_approved_dry" : "media.toast_approved", record.title))
        }
    }

    private func run(_ work: () async throws -> Void) async {
        do { try await work() } catch { toast.show(APIError.from(error).localizedMessage, kind: .error) }
    }
}

struct RequestRow: View {
    let record: RequestRecord
    var showRequester = false
    var busy = false
    /// Newer progress pushed by the server.
    var progress: LiveProgress?
    /// A tappable link to the detail (stack); off in the two-column layout where the list selection does it.
    var link = true
    var onApprove: () -> Void = {}
    var onDecline: () -> Void = {}

    var body: some View {
        if link {
            RequestLink(record: record) { rowContent }
        } else {
            rowContent
        }
    }

    private var rowContent: some View {
        HStack(alignment: .top, spacing: Tokens.Spacing.md) {
            RemoteImage(path: record.posterPath, size: .w185) {
                Image(systemName: "film").foregroundStyle(Tokens.palette.mutedFg)
            }
            .frame(width: 64, height: 96)
            .background(Tokens.palette.muted)
            .clipShape(.rect(cornerRadius: Tokens.Radius.sm))
            .accessibilityHidden(true)
            VStack(alignment: .leading, spacing: Tokens.Spacing.xs) {
                HStack(alignment: .top) {
                    Text(verbatim: record.title).font(.body.weight(.semibold)).lineLimit(2)
                    Spacer(minLength: Tokens.Spacing.sm)
                    StatusBadge(state: record.state)
                }
                Text(choose(record.type == .tv, "type.tv", "type.movie")).font(.footnote).foregroundStyle(Tokens.palette.mutedFg)
                Text(verbatim: when).font(.caption).foregroundStyle(Tokens.palette.mutedFg)
                if record.state == .downloading, let percent = progress?.percent ?? record.progressPercent {
                    ProgressBar(percent: percent).padding(.top, Tokens.Spacing.xs)
                }
                if showRequester && record.state == .requested {
                    HStack(spacing: Tokens.Spacing.sm) {
                        Button(action: onApprove) {
                            HStack(spacing: Tokens.Spacing.xs) { Image(systemName: "checkmark"); Text("req.approve") }
                        }
                        .buttonStyle(.tipsarr(.primary, compact: true))
                        Button(action: onDecline) {
                            HStack(spacing: Tokens.Spacing.xs) { Image(systemName: "xmark"); Text("req.decline") }
                        }
                        .buttonStyle(.tipsarr(.secondary, compact: true))
                    }
                    .disabled(busy)
                    .padding(.top, Tokens.Spacing.xs)
                }
            }
        }
        .opacity(busy ? 0.6 : 1)
    }

    private var when: String {
        let ago = record.createdAt.formatted(.relative(presentation: .named))
        if showRequester, let name = record.requestedBy, !name.isEmpty { return L10n.string("m.requests.by_ago", name, ago) }
        return L10n.string("m.requests.requested_ago", ago)
    }
}

struct DeclineSheet: View {
    let title: String
    var onDecline: (String) async throws -> Void

    @Environment(\.dismiss) private var dismiss
    @State private var reason = ""
    @State private var sending = false
    @State private var errorText: String?

    var body: some View {
        VStack(alignment: .leading, spacing: Tokens.Spacing.lg) {
            Text(verbatim: L10n.string("requests.decline_title", title)).font(.title3.weight(.semibold))
            TextField("requests.decline_placeholder", text: $reason, axis: .vertical)
                .lineLimit(3...6)
                .padding(Tokens.Spacing.md)
                .background(Tokens.palette.muted, in: .rect(cornerRadius: Tokens.Radius.md))
            if let errorText { Banner(kind: .error, title: LText.verbatim(errorText)) }
            VStack(spacing: Tokens.Spacing.sm) {
                Button { Task { await send() } } label: {
                    Label { Text("m.requests.decline_send") } icon: { Image(systemName: "xmark") }
                }
                .buttonStyle(.tipsarr(.destructive, fullWidth: true))
                .disabled(sending)
                Button("common.cancel") { dismiss() }.buttonStyle(.tipsarr(.ghost, fullWidth: true))
            }
            Spacer(minLength: 0)
        }
        .padding(Tokens.Spacing.xl)
        .padding(.top, Tokens.Spacing.md)
    }

    private func send() async {
        sending = true
        errorText = nil
        defer { sending = false }
        do { try await onDecline(reason.trimmingCharacters(in: .whitespacesAndNewlines)) } catch { errorText = APIError.from(error).localizedMessage }
    }
}
