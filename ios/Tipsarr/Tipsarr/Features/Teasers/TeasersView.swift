import AVFoundation
import SwiftUI

/// The Teasers tab: one trailer per page, swipe up for the next one.
struct TeasersView: View {
    /// This tab is the selected one (wide windows keep every opened tab alive).
    let isSelected: Bool
    @Binding var path: NavigationPath
    /// Height of our own tab bar when the shell draws one over the content (iPad, Duo); the system bar is already in the safe area.
    var bottomInset: CGFloat = 0
    /// The server answered "teasers_disabled": the feature was turned off while the tab was open.
    var onUnavailable: () -> Void = {}

    @State private var model: TeasersModel
    @State private var current: String?
    @State private var tapPaused = false
    /// Pinched in: the picture fills the screen and most of the interface is hidden.
    @State private var zoomed = false
    /// Pinching hid the interface: it stays hidden until the next trailer or the button that brings it back.
    @State private var uiHidden = false
    @State private var landscape = false
    @State private var draft: RequestDraft?
    @State private var pool: TeaserVideoPool
    /// Sound stays off for later trailers and launches once the person muted a trailer.
    @AppStorage("teasersMuted") private var muted = false
    @Environment(\.scenePhase) private var scenePhase
    @Environment(\.appContext) private var context
    @Environment(\.detailPane) private var pane
    @Environment(ToastCenter.self) private var toast

    private static let endID = "teasers-end"
    private static let waitID = "teasers-wait"

    init(api: TipsarrAPI, isSelected: Bool, path: Binding<NavigationPath>, bottomInset: CGFloat = 0, onUnavailable: @escaping () -> Void = {}) {
        _model = State(initialValue: TeasersModel(api: api))
        _pool = State(initialValue: TeaserVideoPool(token: api.token))
        self.onUnavailable = onUnavailable
        self.isSelected = isSelected
        _path = path
        self.bottomInset = bottomInset
    }

    /// The waiting page is on screen (or is the only page): trailers are being prepared.
    private var waiting: Bool {
        guard isSelected, !model.done else { return false }
        return model.items.isEmpty || current == Self.waitID
    }

    /// Only the visible page plays, and only while the tab is shown, the app is in front and nothing covers the feed.
    private var feedActive: Bool { isSelected && scenePhase == .active && path.isEmpty && draft == nil }

    var body: some View {
        NavigationStack(path: $path) {
            GeometryReader { geo in
                // The feed fills the whole screen; `insets` tell the cards where the notch, the home bar and the tab bar are.
                let insets = geo.safeAreaInsets
                content(insets: insets, width: geo.size.width)
                    .frame(width: geo.size.width + insets.leading + insets.trailing, height: geo.size.height + insets.top + insets.bottom)
                    .offset(x: -insets.leading, y: -insets.top)
            }
            .onGeometryChange(for: Bool.self, of: { $0.size.width > $0.size.height }) { landscape = $0 }
            .background(Color.black)
            .toolbar(.hidden, for: .navigationBar)
            // The tab bar gives way in landscape.
            .toolbarVisibility(landscape || (uiHidden && !zoomed) ? .hidden : .automatic, for: .tabBar)
            .mediaDestinations()
        }
        .task { await model.loadIfNeeded() }
        .onChange(of: model.phase) { _, phase in
            if case .failed(let error) = phase, error.code == "teasers_disabled" {
                toast.show(L10n.string("m.teasers.error"), kind: .error)
                onUnavailable()
            }
        }
        .onChange(of: current) { _, _ in preload() }
        .onChange(of: model.items.count) { _, _ in preload() }
        .onChange(of: isSelected, initial: true) { _, selected in
            // Playback follows the media volume, not the ringer switch.
            let session = AVAudioSession.sharedInstance()
            if selected {
                try? session.setCategory(.playback, mode: .moviePlayback)
                try? session.setActive(true)
            } else {
                zoomed = false
                uiHidden = false
                try? session.setActive(false, options: .notifyOthersOnDeactivation)
            }
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
                        teaser: teaser, video: pool.video(for: teaser), isCurrent: current == teaser.id || (current == nil && teaser.id == model.items.first?.id),
                        feedActive: feedActive, paused: tapPaused, muted: muted, uiHidden: uiHidden, bottomInset: max(insets.bottom, bottomInset), topInset: insets.top,
                        leadingInset: insets.leading, trailingInset: insets.trailing, showsHint: model.items.first?.id == teaser.id,
                        onTap: { tapPaused.toggle() },
                        onZoom: { zoomed = $0; if $0 { uiHidden = true } },
                        onShowUI: { uiHidden = false },
                        onMute: { muted.toggle() },
                        onRequest: { Task { await request(teaser) } },
                        onDetails: { open(teaser) },
                        onWatchlist: { Task { await run { try await model.toggleWatchlist(teaser.id) } } },
                        onHide: { Task { await hide(teaser) } },
                        onFileFailed: { skip(teaser.id) }
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
                } else {
                    // The next trailers are still downloading on the server: this page waits for them.
                    VStack(spacing: Tokens.Spacing.lg) {
                        ProgressView().tint(.white)
                        Text("m.teasers.preparing").font(.subheadline).foregroundStyle(.white.opacity(0.8)).multilineTextAlignment(.center)
                    }
                    .padding(.horizontal, Tokens.Spacing.xl)
                    .containerRelativeFrame([.horizontal, .vertical])
                    .id(Self.waitID)
                }
            }
            .scrollTargetLayout()
        }
        .scrollTargetBehavior(.paging)
        .scrollPosition(id: $current)
        .scrollIndicators(.hidden)
        .scrollDisabled(zoomed)
        // A phone-shaped column on wide windows (tablets), never on a phone turned sideways.
        .frame(maxWidth: width > 700 && UIDevice.current.userInterfaceIdiom != .phone ? 520 : .infinity)
        .frame(maxWidth: .infinity)
        .task(id: waiting) {
            // Ask again every few seconds while the person looks at the waiting page.
            while waiting, !Task.isCancelled {
                if let first = await model.loadMore() { current = first; break }
                try? await Task.sleep(for: .seconds(8))
            }
        }
        .onChange(of: current) { _, id in
            tapPaused = false
            uiHidden = false
            if id == Self.endID { model.reachedEnd() } else if id != Self.waitID { Task { await model.visible(id) } }
        }
    }

    // MARK: Actions

    /// The current card and the next one get their video ready (the next one buffers while paused); the rest is released.
    private func preload() {
        guard isSelected else { return }
        let items = model.items
        guard let index = items.firstIndex(where: { $0.id == current }) ?? (current == nil ? 0 : nil), !items.isEmpty else { return }
        let wanted = items[index...].prefix(2)
        pool.keep(Set(wanted.map(\.id)))
        for teaser in wanted { _ = pool.video(for: teaser) }
    }

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
    /// The native player of this card.
    let video: TeaserVideo
    let isCurrent: Bool
    let feedActive: Bool
    let paused: Bool
    let muted: Bool
    let uiHidden: Bool
    let bottomInset: CGFloat
    let topInset: CGFloat
    /// Sideways, the notch side: the title and description keep clear of it, the action buttons do not.
    let leadingInset: CGFloat
    let trailingInset: CGFloat
    let showsHint: Bool
    var onTap: () -> Void
    var onZoom: (Bool) -> Void
    var onShowUI: () -> Void
    var onMute: () -> Void
    var onRequest: () -> Void
    var onDetails: () -> Void
    var onWatchlist: () -> Void
    var onHide: () -> Void
    var onFileFailed: () -> Void

    @Environment(\.appContext) private var context
    /// Hold on a side of the page: 2x speed.
    @State private var fast = false
    /// A finger is on the progress bar.
    @State private var scrubbing = false
    @State private var overviewOpen = false
    /// Pinch scale while the fingers are down; it springs back to 1 on release.
    @GestureState private var live = Pinch()

    private struct Pinch: Equatable {
        var scale: CGFloat = 1
        var anchor: UnitPoint = .center
    }

    /// Fingers are pinching: the interface is hidden.
    private var zoomed: Bool { live.scale > 1.01 }
    private var hidden: Bool { uiHidden || zoomed }

    private var playing: Bool { isCurrent && feedActive }
    private var wantsPlay: Bool { playing && !paused }
    private var loopCount: Int { video.loops }

    var body: some View {
        ZStack {
            Color.black
            RemoteImage(path: teaser.item.backdropPath ?? teaser.item.posterPath, size: .w780) { Color.black }
                .opacity(0.55)
                .blur(radius: 24)
            TeaserVideoView(video: video)
                .id(ObjectIdentifier(video))
                .scaleEffect(live.scale, anchor: live.anchor)
                .animation(live.scale == 1 ? .spring(duration: 0.3) : nil, value: live.scale)
                .ignoresSafeArea()
                .opacity(video.isReady ? 1 : 0)
                .animation(.easeIn(duration: 0.2), value: video.isReady)
                .onChange(of: wantsPlay, initial: true) { _, play in video.set(playing: play, muted: muted) }
                .onChange(of: muted) { _, now in video.set(playing: wantsPlay, muted: now) }
                .onChange(of: fast) { _, now in video.setFast(now) }
                .onChange(of: video.failed) { _, failed in if failed { onFileFailed() } }
            if playing, !video.isReady, !paused { ProgressView().tint(.white).allowsHitTesting(false) }
            touchLayer
            if paused && playing {
                Image(systemName: "play.fill").font(.system(size: 56)).foregroundStyle(.white.opacity(0.85)).allowsHitTesting(false)
            }
            if !hidden {
                LinearGradient(colors: [.clear, .black.opacity(0.75)], startPoint: .center, endPoint: .bottom).allowsHitTesting(false)
                overlay
            }
            if video.isReady, !hidden { scrubBar(video) }
            if fast { speedBadge }
            if uiHidden && !zoomed { showUIButton }
        }
        .clipped()
        .simultaneousGesture(
            MagnifyGesture()
                .updating($live) { value, state, _ in
                    guard isCurrent else { return }
                    state = Pinch(scale: min(max(value.magnification, 1), 4), anchor: value.startAnchor)
                }
        )
        // A gesture state resets by itself when the fingers lift or the system cancels the gesture.
        .onChange(of: zoomed) { _, now in onZoom(now) }
        .onChange(of: isCurrent) { _, current in if !current { fast = false; scrubbing = false } }
    }

    /// Tap anywhere pauses; holding on a side (like TikTok) plays at 2x until the finger lifts.
    private var touchLayer: some View {
        GeometryReader { geo in
            HStack(spacing: 0) {
                edge.frame(width: geo.size.width * 0.28)
                Color.clear.contentShape(.rect).onTapGesture(perform: onTap)
                edge.frame(width: geo.size.width * 0.28)
            }
        }
    }

    private var edge: some View {
        Color.clear.contentShape(.rect)
            .onTapGesture(perform: onTap)
            .onLongPressGesture(minimumDuration: 0.25, maximumDistance: 60, perform: { fast = true }, onPressingChanged: { pressing in
                if !pressing { fast = false }
            })
    }

    /// Brings the interface back after a zoom hid it.
    private var showUIButton: some View {
        VStack {
            HStack {
                Spacer()
                Button(action: onShowUI) {
                    Image(systemName: "eye")
                        .font(.body.weight(.semibold))
                        .foregroundStyle(.white)
                        .frame(width: 44, height: 44)
                        .background(.black.opacity(0.45), in: .circle)
                        .contentShape(.circle)
                }
                .buttonStyle(.plain)
                .accessibilityLabel(Text("m.teasers.show_interface"))
            }
            .padding(.top, topInset + Tokens.Spacing.md)
            .padding(.trailing, Tokens.Spacing.lg)
            Spacer()
        }
    }

    private var speedBadge: some View {
        VStack {
            Label { Text(verbatim: "2×") } icon: { Image(systemName: "forward.fill") }
                .font(.subheadline.weight(.bold))
                .foregroundStyle(.white)
                .padding(.horizontal, Tokens.Spacing.md).padding(.vertical, Tokens.Spacing.sm)
                .background(.black.opacity(0.5), in: .capsule)
                .padding(.top, topInset + Tokens.Spacing.lg)
            Spacer()
        }
        .allowsHitTesting(false)
    }

    /// The progress line along the bottom edge: drag it to go forward or back.
    private func scrubBar(_ video: TeaserVideo) -> some View {
        VStack {
            Spacer()
            GeometryReader { geo in
                ZStack(alignment: .bottomLeading) {
                    Color.clear
                    Rectangle().fill(.white.opacity(0.25)).frame(height: scrubbing ? 8 : 3)
                    Rectangle().fill(.white.opacity(0.9)).frame(width: geo.size.width * video.progress, height: scrubbing ? 8 : 3)
                }
                .contentShape(.rect)
                .animation(.easeOut(duration: 0.12), value: scrubbing)
                .gesture(
                    DragGesture(minimumDistance: 0)
                        .onChanged { drag in
                            if !scrubbing { scrubbing = true; video.beginScrub() }
                            video.scrub(to: drag.location.x / max(geo.size.width, 1))
                        }
                        .onEnded { _ in
                            scrubbing = false
                            video.endScrub()
                        }
                )
            }
            .frame(height: 28)
            .padding(.bottom, bottomInset)
        }
    }

    private static func clock(_ seconds: Double) -> String {
        let total = max(Int(seconds.rounded(.down)), 0)
        return String(format: "%d:%02d", total / 60, total % 60)
    }

    private var overlay: some View {
        VStack {
            Spacer()
            HStack(alignment: .bottom, spacing: Tokens.Spacing.lg) {
                info
                actions
            }
            .padding(.leading, Tokens.Spacing.lg + leadingInset)
            .padding(.trailing, Tokens.Spacing.lg)
            .padding(.bottom, max(bottomInset, Tokens.Spacing.lg) + Tokens.Spacing.md + 24)
        }
        .environment(\.colorScheme, .dark)
    }

    private var info: some View {
        VStack(alignment: .leading, spacing: Tokens.Spacing.sm) {
            Button(action: onDetails) {
                Text(verbatim: teaser.item.title)
                    .font(.title2.weight(.bold))
                    .lineLimit(2)
                    .multilineTextAlignment(.leading)
                    .contentShape(.rect)
            }
            .buttonStyle(.plain)
            HStack(spacing: Tokens.Spacing.sm) {
                Text(teaser.item.type == .tv ? "m.poster.tv" : "m.poster.movie")
                if let year = teaser.item.releaseYear { Text(verbatim: "· \(year)") }
                if teaser.item.voteAverage > 0 {
                    Text(verbatim: "·")
                    RatingLabel(input: RatingInput(type: teaser.item.type, tmdbId: teaser.item.tmdbId, tmdb: teaser.item.voteAverage), textColor: .white.opacity(0.85))
                }
            }
            .font(.subheadline.weight(.medium))
            .foregroundStyle(.white.opacity(0.85))
            if scrubbing {
                Text(verbatim: "\(Self.clock(video.currentTime)) / \(Self.clock(video.duration))")
                    .font(.title3.weight(.semibold).monospacedDigit())
            } else if let overview = teaser.item.overview {
                overviewText(overview)
            }
            if showsHint, loopCount >= 2 {
                Label { Text("m.teasers.swipe_hint") } icon: { Image(systemName: "chevron.up") }
                    .font(.footnote).foregroundStyle(.white.opacity(0.7)).padding(.top, Tokens.Spacing.xs)
            }
        }
        .foregroundStyle(.white)
        .frame(maxWidth: .infinity, alignment: .leading)
        .shadow(color: .black.opacity(0.5), radius: 4)
    }

    /// Two lines; a tap shows all of it (scrollable when long) and a second tap folds it back.
    @ViewBuilder private func overviewText(_ overview: String) -> some View {
        let text = Text(verbatim: overview).font(.subheadline).foregroundStyle(.white.opacity(0.85)).multilineTextAlignment(.leading)
        Group {
            if overviewOpen {
                ScrollView { text.frame(maxWidth: .infinity, alignment: .leading) }
                    .frame(maxHeight: 180)
                    .scrollIndicators(.hidden)
            } else {
                text.lineLimit(2)
            }
        }
        .contentShape(.rect)
        .onTapGesture { withAnimation(.easeInOut(duration: 0.2)) { overviewOpen.toggle() } }
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
