import SwiftUI

/// Titles saved with "Watchlist". Long press removes one.
struct WatchlistScreen: View {
    @Environment(\.appContext) private var context
    @Environment(ToastCenter.self) private var toast
    @State private var items: [MediaItem] = []
    @State private var phase: Phase = .loading

    enum Phase: Equatable { case loading, loaded, failed(APIError) }

    var body: some View {
        TitleGridScreen(title: "watchlist.title", hint: nil, items: items, phase: phase, emptyMessage: "watchlist.empty",
                        actionTitle: "m.library.remove", actionSymbol: "bookmark.slash",
                        reload: load) { item in
            guard let context else { return }
            do {
                try await context.api.setWatchlisted(false, item.type, id: item.tmdbId)
                items.removeAll { $0.id == item.id }
                toast.show(L10n.string("actions.toast_watch_removed"))
            } catch {
                toast.show(APIError.from(error).localizedMessage, kind: .error)
            }
        }
        .task { await load() }
    }

    private func load() async {
        guard let context else { return }
        do {
            items = try await context.api.watchlist()
            phase = .loaded
        } catch {
            if items.isEmpty { phase = .failed(APIError.from(error)) }
        }
    }
}

/// Titles marked "not interested". Long press shows one again.
struct HiddenScreen: View {
    @Environment(\.appContext) private var context
    @Environment(ToastCenter.self) private var toast
    @State private var items: [MediaItem] = []
    @State private var phase: WatchlistScreen.Phase = .loading

    var body: some View {
        TitleGridScreen(title: "profile.hidden_title", hint: "profile.hidden_desc", items: items, phase: phase, emptyMessage: "profile.nothing_hidden",
                        actionTitle: "m.library.show_again", actionSymbol: "eye",
                        reload: load) { item in
            guard let context else { return }
            do {
                try await context.api.setBlocklisted(false, item.type, id: item.tmdbId)
                items.removeAll { $0.id == item.id }
                toast.show(L10n.string("actions.toast_unhidden"))
            } catch {
                toast.show(APIError.from(error).localizedMessage, kind: .error)
            }
        }
        .task { await load() }
    }

    private func load() async {
        guard let context else { return }
        do {
            items = try await context.api.blocklist()
            phase = .loaded
        } catch {
            if items.isEmpty { phase = .failed(APIError.from(error)) }
        }
    }
}

private struct TitleGridScreen: View {
    let title: LText
    let hint: LText?
    let items: [MediaItem]
    let phase: WatchlistScreen.Phase
    let emptyMessage: LText
    let actionTitle: LText
    let actionSymbol: String
    let reload: () async -> Void
    let onAction: (MediaItem) async -> Void

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: Tokens.Spacing.lg) {
                if let hint { Text(hint).font(.subheadline).foregroundStyle(Tokens.palette.mutedFg).padding(.horizontal, Tokens.Spacing.lg) }
                switch phase {
                case .loading:
                    LazyVGrid(columns: Self.columns, spacing: Tokens.Spacing.lg) { ForEach(0..<6, id: \.self) { _ in PosterSkeleton() } }
                        .padding(.horizontal, Tokens.Spacing.lg)
                case .failed(let error):
                    ErrorState(error: error) { await reload() }
                case .loaded where items.isEmpty:
                    StateView(symbol: "tray", title: emptyMessage).frame(minHeight: 320)
                case .loaded:
                    LazyVGrid(columns: Self.columns, spacing: Tokens.Spacing.lg) {
                        ForEach(items) { item in
                            NavigationLink(value: item.route) {
                                PosterCard(item: item)
                            }
                            .buttonStyle(.plain)
                            .contextMenu {
                                Button { Task { await onAction(item) } } label: { Label { Text(actionTitle) } icon: { Image(systemName: actionSymbol) } }
                            }
                        }
                    }
                    .padding(.horizontal, Tokens.Spacing.lg)
                }
            }
            .padding(.vertical, Tokens.Spacing.sm)
        }
        .background(Tokens.palette.bg)
        .navigationTitle(Text(title))
        .navigationBarTitleDisplayMode(.inline)
        .refreshable { await reload() }
    }

    private static let columns = [GridItem(.adaptive(minimum: 104, maximum: 180), spacing: Tokens.Spacing.md, alignment: .top)]
}
