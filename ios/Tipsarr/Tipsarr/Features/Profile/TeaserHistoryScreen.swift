import SwiftUI

/// What the person swiped past in Teasers. Those titles stay out of the feed until they are forgotten here.
struct TeaserHistoryScreen: View {
    /// Another person whose history an administrator looks at; nil is the signed-in person's own.
    struct Owner: Hashable { var id: String; var name: String }
    var owner: Owner?

    @Environment(\.appContext) private var context
    @Environment(ToastCenter.self) private var toast
    @State private var entries: [TeaserHistoryEntry] = []
    @State private var total = 0
    @State private var phase: WatchlistScreen.Phase = .loading
    @State private var loadingMore = false
    @State private var confirmWipe = false

    var body: some View {
        Group {
            switch phase {
            case .loading:
                ProgressView().frame(maxWidth: .infinity, maxHeight: .infinity)
            case .failed(let error):
                StateView(symbol: "exclamationmark.triangle", title: "m.teasers.error", message: LText.verbatim(error.localizedMessage),
                          actionTitle: "common.retry") { Task { await load() } }
            case .loaded where entries.isEmpty:
                StateView(symbol: "clock.arrow.circlepath", title: "m.teasers.history.empty")
            case .loaded:
                list
            }
        }
        .background(Tokens.palette.bg)
        .navigationTitle(Text(verbatim: owner.map { L10n.string("m.users.teasers_history_title", $0.name) } ?? L10n.string("m.teasers.history.title")))
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            if !entries.isEmpty {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("m.teasers.history.forget_all", role: .destructive) { confirmWipe = true }
                }
            }
        }
        .confirmationDialog(Text(verbatim: owner.map { L10n.string("m.users.teasers_history_confirm_all", $0.name) } ?? L10n.string("m.teasers.history.confirm_all")), isPresented: $confirmWipe, titleVisibility: .visible) {
            Button("m.teasers.history.forget_all", role: .destructive) { Task { await wipe() } }
        }
        .task { await load() }
        .refreshable { await load() }
    }

    private var list: some View {
        List {
            Section {
                ForEach(entries) { entry in
                    MediaLink(route: entry.item.route) { row(entry) }
                        .buttonStyle(.plain)
                        .swipeActions(edge: .trailing) {
                            Button { Task { await forget(entry) } } label: {
                                Label("m.teasers.history.forget", systemImage: "arrow.uturn.backward")
                            }
                            .tint(Tokens.palette.primary)
                        }
                        .onAppear { if entry.id == entries.last?.id { Task { await loadMore() } } }
                }
            } header: {
                Text("m.teasers.history.hint").textCase(nil).font(.footnote)
            }
        }
        .listStyle(.insetGrouped)
        .scrollContentBackground(.hidden)
    }

    private func row(_ entry: TeaserHistoryEntry) -> some View {
        HStack(spacing: Tokens.Spacing.md) {
            RemoteImage(path: entry.item.posterPath, size: .w185) { Image(systemName: "film").foregroundStyle(Tokens.palette.mutedFg) }
                .frame(width: 44, height: 66)
                .background(Tokens.palette.muted)
                .clipShape(.rect(cornerRadius: Tokens.Radius.sm))
            VStack(alignment: .leading, spacing: 2) {
                Text(verbatim: entry.item.title).font(.body.weight(.medium)).lineLimit(2)
                Text(verbatim: L10n.string("m.teasers.history.seen_on", entry.seenAt.formatted(.relative(presentation: .named))))
                    .font(.footnote).foregroundStyle(Tokens.palette.mutedFg)
            }
            Spacer(minLength: 0)
        }
        .contentShape(.rect)
    }

    private func load() async {
        guard let context else { return }
        do {
            let page = try await context.api.teaserHistory(offset: 0, owner: owner?.id)
            entries = page.entries
            total = page.total
            phase = .loaded
        } catch {
            if entries.isEmpty { phase = .failed(APIError.from(error)) }
        }
    }

    /// The last row is on screen: fetch the next page (a page can hold fewer titles than asked when TMDB no longer knows some).
    private func loadMore() async {
        guard let context, !loadingMore, entries.count < total else { return }
        loadingMore = true
        defer { loadingMore = false }
        guard let page = try? await context.api.teaserHistory(offset: entries.count, owner: owner?.id) else { return }
        let known = Set(entries.map(\.id))
        entries += page.entries.filter { !known.contains($0.id) }
        total = page.total
        if page.entries.isEmpty { total = entries.count }
    }

    private func forget(_ entry: TeaserHistoryEntry) async {
        guard let context else { return }
        do {
            try await context.api.forgetTeaser(entry.item.type, id: entry.item.tmdbId, owner: owner?.id)
            entries.removeAll { $0.id == entry.id }
            total = max(total - 1, 0)
            toast.show(L10n.string("m.teasers.history.forgot", entry.item.title))
        } catch {
            toast.show(APIError.from(error).localizedMessage, kind: .error)
        }
    }

    private func wipe() async {
        guard let context else { return }
        do {
            try await context.api.forgetAllTeasers(owner: owner?.id)
            entries = []
            total = 0
            toast.show(owner.map { L10n.string("m.users.teasers_history_wiped", $0.name) } ?? L10n.string("m.teasers.history.wiped"))
        } catch {
            toast.show(APIError.from(error).localizedMessage, kind: .error)
        }
    }
}
