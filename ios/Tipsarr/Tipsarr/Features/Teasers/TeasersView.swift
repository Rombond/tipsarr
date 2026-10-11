import AVFoundation
import SwiftUI

/// The Teasers tab: one trailer per page, swipe up for the next one.
struct TeasersView: View {
    /// This tab is the selected one (wide windows keep every opened tab alive).
    let isSelected: Bool
    @Binding var path: NavigationPath
    /// Height of our own tab bar when the shell draws one over the content (iPad, Duo); the system bar is already in the safe area.
    var bottomInset: CGFloat = 0

    @State private var model: TeasersModel
    @State private var current: String?
    @State private var tapPaused = false
    @State private var draft: RequestDraft?
    /// Sound stays off for later trailers and launches once the person muted a trailer.
    @AppStorage("teasersMuted") private var muted = false
    @Environment(\.scenePhase) private var scenePhase
    @Environment(\.appContext) private var context
    @Environment(\.detailPane) private var pane
    @Environment(ToastCenter.self) private var toast

    private static let endID = "teasers-end"

    init(api: TipsarrAPI, isSelected: Bool, path: Binding<NavigationPath>, bottomInset: CGFloat = 0) {
        _model = State(initialValue: TeasersModel(api: api))
        self.isSelected = isSelected
        _path = path
        self.bottomInset = bottomInset
    }

    /// Only the visible page plays, and only while the tab is shown, the app is in front and nothing covers the feed.
    private var feedActive: Bool { isSelected && scenePhase == .active && path.isEmpty && draft == nil }

    var body: some View {
        NavigationStack(path: $path) {
            GeometryReader { geo in
                content(insets: geo.safeAreaInsets, width: geo.size.width)
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            }
            .background(Color.black)
            .toolbar(.hidden, for: .navigationBar)
            .mediaDestinations()
        }
        .task { await model.loadIfNeeded() }
        .onChange(of: isSelected, initial: true) { _, selected in
            // The ambient category follows the silent switch and leaves the person's music alone.
            if selected { try? AVAudioSession.sharedInstance().setCategory(.ambient) }
        }
        .sheet(item: $draft) { draft in
            RequestSheet(model: draft.model) { record in
                self.draft = nil
                model.setState(record.state, for: draft.teaserID)
                announce(record, title: draft.model.route.title)
            }
            .tipsarrSheet(detents: nil)
        }
    }

    // MARK: Pages

    @ViewBuilder private func content(insets: EdgeInsets, width: CGFloat) -> some View {
        switch model.phase {
        case .loading where model.items.isEmpty:
            ProgressView().tint(.white).frame(maxWidth: .infinity, maxHeight: .infinity)
        case .failed(let error) where model.items.isEmpty:
            StateView(symbol: "exclamationmark.triangle", title: "m.teasers.error", message: LText.verbatim(error.localizedMessage),
                      actionTitle: "common.retry") { Task { await model.load() } }
                .background(Tokens.palette.bg)
        default:
            pager(insets: insets, width: width)
        }
    }

    private func pager(insets: EdgeInsets, width: CGFloat) -> some View {
        ScrollView(.vertical) {
            LazyVStack(spacing: 0) {
                ForEach(model.items) { teaser in
                    TeaserPage(
                        teaser: teaser, isCurrent: current == teaser.id || (current == nil && teaser.id == model.items.first?.id),
                        feedActive: feedActive, paused: tapPaused, muted: muted, bottomInset: max(insets.bottom, bottomInset), showsHint: model.items.first?.id == teaser.id,
                        onTap: { tapPaused.toggle() },
                        onMute: { muted.toggle() },
                        onRequest: { Task { await request(teaser) } },
                        onDetails: { open(teaser) },
                        onWatchlist: { Task { await run { try await model.toggleWatchlist(teaser.id) } } },
                        onHide: { Task { await hide(teaser) } },
                        onPlayerError: { code in if [100, 101, 150, 152].contains(code) { skip(teaser.id) } }
                    )
                    .containerRelativeFrame([.horizontal, .vertical])
                    .id(teaser.id)
                }
                if model.done {
                    StateView(symbol: "checkmark.circle", title: "m.teasers.done")
                        .padding(.horizontal, Tokens.Spacing.xl)
                        .background(Tokens.palette.bg)
                        .containerRelativeFrame([.horizontal, .vertical])
                        .id(Self.endID)
                }
            }
            .scrollTargetLayout()
        }
        .scrollTargetBehavior(.paging)
        .scrollPosition(id: $current)
        .scrollIndicators(.hidden)
        // A phone-shaped column on wide windows.
        .frame(maxWidth: width > 700 ? 520 : .infinity)
        .frame(maxWidth: .infinity)
        .ignoresSafeArea()
        .onChange(of: current) { _, id in
            tapPaused = false
            if id == Self.endID { model.reachedEnd() } else { Task { await model.visible(id) } }
        }
    }

    // MARK: Actions

    private func open(_ teaser: Teaser) {
        if let pane { pane.content = .media(teaser.item.route) } else { path.append(teaser.item.route) }
    }

    /// Takes a card out and keeps the next one in view.
    private func skip(_ id: String) {
        advance(from: id)
        model.remove(id)
    }

    private func advance(from id: String) {
        guard current == id || (current == nil && model.items.first?.id == id), let index = model.items.firstIndex(where: { $0.id == id }) else { return }
        let next = model.items.indices.contains(index + 1) ? model.items[index + 1].id : (model.done ? Self.endID : nil)
        current = next ?? (index > 0 ? model.items[index - 1].id : nil)
    }

    private func hide(_ teaser: Teaser) async {
        advance(from: teaser.id)
        await run { try await model.hide(teaser.id) }
    }

    private func run(_ work: () async throws -> Void) async {
        do { try await work() } catch { toast.show(APIError.from(error).localizedMessage, kind: .error) }
    }

    private func request(_ teaser: Teaser) async {
        guard let context else { return }
        let detail = MediaDetailModel(route: teaser.item.route, context: context)
        await detail.load()
        if await detail.prepareRequest() {
            draft = RequestDraft(model: detail, teaserID: teaser.id)
            return
        }
        do {
            let seasons = teaser.item.type == .tv ? detail.regularSeasons.map(\.number) : nil
            let record = try await detail.submitRequest(seasons: seasons, profileID: nil, folder: nil)
            model.setState(record.state, for: teaser.id)
            announce(record, title: teaser.item.title)
        } catch {
            toast.show(APIError.from(error).localizedMessage, kind: .error)
        }
    }

    private func announce(_ record: RequestRecord, title: String) {
        let key = record.state == .requested ? "media.toast_requested" : (record.dryRun ? "media.toast_approved_dry" : "media.toast_approved")
        toast.show(L10n.string(key, title))
    }
}

/// A request that needs the sheet (quality profile or seasons to choose).
private struct RequestDraft: Identifiable {
    let id = UUID()
    let model: MediaDetailModel
    let teaserID: String
}

// MARK: Page

private struct TeaserPage: View {
    let teaser: Teaser
    let isCurrent: Bool
    let feedActive: Bool
    let paused: Bool
    let muted: Bool
    let bottomInset: CGFloat
    let showsHint: Bool
    var onTap: () -> Void
    var onMute: () -> Void
    var onRequest: () -> Void
    var onDetails: () -> Void
    var onWatchlist: () -> Void
    var onHide: () -> Void
    var onPlayerError: (Int) -> Void

    @Environment(\.appContext) private var context

    private var playing: Bool { isCurrent && feedActive }

    var body: some View {
        ZStack {
            Color.black
            RemoteImage(path: teaser.item.backdropPath ?? teaser.item.posterPath, size: .w780) { Color.black }
                .opacity(0.55)
                .blur(radius: 24)
            if playing {
                TeaserPlayer(videoID: teaser.trailerKey, muted: muted, playing: !paused, onError: onPlayerError)
                    .id(teaser.trailerKey)
                    .transition(.opacity)
            }
            Color.clear.contentShape(.rect).onTapGesture(perform: onTap)
            if paused && playing {
                Image(systemName: "play.fill").font(.system(size: 56)).foregroundStyle(.white.opacity(0.85)).allowsHitTesting(false)
            }
            LinearGradient(colors: [.clear, .black.opacity(0.75)], startPoint: .center, endPoint: .bottom).allowsHitTesting(false)
            overlay
        }
        .clipped()
    }

    private var overlay: some View {
        VStack {
            Spacer()
            HStack(alignment: .bottom, spacing: Tokens.Spacing.lg) {
                info
                actions
            }
            .padding(.horizontal, Tokens.Spacing.lg)
            .padding(.bottom, max(bottomInset, Tokens.Spacing.lg) + Tokens.Spacing.md)
        }
        .environment(\.colorScheme, .dark)
    }

    private var info: some View {
        VStack(alignment: .leading, spacing: Tokens.Spacing.sm) {
            Text(verbatim: teaser.item.title)
                .font(.title2.weight(.bold))
                .lineLimit(2)
            HStack(spacing: Tokens.Spacing.sm) {
                Text(teaser.item.type == .tv ? "m.poster.tv" : "m.poster.movie")
                if let year = teaser.item.releaseYear { Text(verbatim: "· \(year)") }
                if teaser.item.voteAverage > 0 {
                    Text(verbatim: "· ★ \(teaser.item.voteAverage.formatted(.number.precision(.fractionLength(1))))")
                }
            }
            .font(.subheadline.weight(.medium))
            .foregroundStyle(.white.opacity(0.85))
            if let overview = teaser.item.overview {
                Text(verbatim: overview).font(.subheadline).lineLimit(2).foregroundStyle(.white.opacity(0.85))
            }
            if showsHint {
                Label { Text("m.teasers.swipe_hint") } icon: { Image(systemName: "chevron.up") }
                    .font(.footnote).foregroundStyle(.white.opacity(0.7)).padding(.top, Tokens.Spacing.xs)
            }
        }
        .foregroundStyle(.white)
        .frame(maxWidth: .infinity, alignment: .leading)
        .shadow(color: .black.opacity(0.5), radius: 4)
    }

    private var actions: some View {
        VStack(spacing: Tokens.Spacing.lg) {
            if let state = teaser.item.state {
                circle(state.symbol, label: state.title, tint: state.color) {}.disabled(true)
            } else if context?.profile.mayRequest(teaser.item.type) ?? false {
                circle("plus", label: "media.request", action: onRequest)
            }
            circle("info.circle", label: "media.view_details", action: onDetails)
            circle(teaser.watchlisted ? "bookmark.fill" : "bookmark", label: teaser.watchlisted ? "actions.on_watchlist" : "actions.watchlist", action: onWatchlist)
            circle("eye.slash", label: "media.not_interested", action: onHide)
            circle(muted ? "speaker.slash.fill" : "speaker.wave.2.fill", label: muted ? "m.teasers.unmute" : "m.teasers.mute", action: onMute)
        }
    }

    private func circle(_ symbol: String, label: LText, tint: Color = .white, action: @escaping () -> Void) -> some View {
        Button(action: action) {
            Image(systemName: symbol)
                .font(.title3.weight(.semibold))
                .foregroundStyle(tint)
                .frame(width: 48, height: 48)
                .background(.black.opacity(0.4), in: .circle)
                .contentShape(.circle)
        }
        .buttonStyle(.plain)
        .accessibilityLabel(Text(label))
    }
}
