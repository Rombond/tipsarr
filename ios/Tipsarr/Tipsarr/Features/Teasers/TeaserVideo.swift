import AVFoundation
import Observation
import SwiftUI

/// One trailer file played natively. The server asks for the same login as the API, so the bearer token goes with every request.
@MainActor @Observable
final class TeaserVideo {
    let player: AVPlayer
    /// 0...1 through the video.
    private(set) var progress: Double = 0
    /// The picture is on screen (the first frame is decoded).
    private(set) var isReady = false
    /// The file could not be loaded or played: the card is dropped from the feed.
    private(set) var failed = false
    /// Landscape trailers are shown whole on the blurred backdrop; a portrait one fills the page.
    private(set) var fillsPage = false
    /// Length of the trailer in seconds (0 until known).
    private(set) var duration: Double = 0
    /// Where the playhead is, in seconds.
    private(set) var currentTime: Double = 0
    /// How many times the trailer played to its end (it loops).
    private(set) var loops = 0
    /// A finger is on the progress bar: the time observer leaves the playhead alone.
    private(set) var scrubbing = false
    private var fast = false

    private var wantPlaying = false
    private var observers: [NSKeyValueObservation] = []
    private var timeObserver: Any?
    private var endObserver: NSObjectProtocol?

    init(url: URL, token: String?) {
        let headers = token.map { ["Authorization": "Bearer \($0)"] } ?? [:]
        let asset = AVURLAsset(url: url, options: ["AVURLAssetHTTPHeaderFieldsKey": headers])
        let item = AVPlayerItem(asset: asset)
        item.preferredForwardBufferDuration = 10
        player = AVPlayer(playerItem: item)
        player.automaticallyWaitsToMinimizeStalling = true
        player.actionAtItemEnd = .none

        observers.append(item.observe(\.status, options: [.initial, .new]) { [weak self] item, _ in
            Task { @MainActor in
                guard let self else { return }
                switch item.status {
                case .failed: self.failed = true
                case .readyToPlay:
                    let size = item.presentationSize
                    self.fillsPage = size.height > 0 && size.width > 0 && size.height >= size.width * 0.9
                    if item.duration.seconds.isFinite { self.duration = item.duration.seconds }
                default: break
                }
            }
        })
        observers.append(player.observe(\.timeControlStatus, options: [.new]) { [weak self] player, _ in
            Task { @MainActor in
                if player.timeControlStatus == .playing { self?.isReady = true }
            }
        })
        timeObserver = player.addPeriodicTimeObserver(forInterval: CMTime(seconds: 0.25, preferredTimescale: 600), queue: .main) { [weak self] time in
            Task { @MainActor in
                guard let self, !self.scrubbing, let duration = self.player.currentItem?.duration.seconds, duration.isFinite, duration > 0 else { return }
                self.duration = duration
                self.currentTime = min(max(time.seconds, 0), duration)
                self.progress = min(max(time.seconds / duration, 0), 1)
            }
        }
        endObserver = NotificationCenter.default.addObserver(forName: .AVPlayerItemDidPlayToEndTime, object: item, queue: .main) { [weak self] _ in
            Task { @MainActor in
                guard let self else { return }
                self.loops += 1
                await self.player.seek(to: .zero)
                if self.wantPlaying { self.player.play() }
            }
        }
    }

    /// Plays or pauses; a paused video keeps buffering, so the next card starts at once.
    func set(playing: Bool, muted: Bool) {
        wantPlaying = playing
        player.isMuted = muted
        if playing { player.play() } else { player.pause() }
    }

    /// Hold-to-speed-up: 2x while a finger rests on the edge of the page.
    func setFast(_ on: Bool) {
        fast = on
        player.defaultRate = on ? 2 : 1
        if wantPlaying && !scrubbing { player.rate = on ? 2 : 1 }
    }

    /// A finger lands on the progress bar: the video pauses and follows it.
    func beginScrub() {
        scrubbing = true
        player.pause()
    }

    /// Moves the playhead to this fraction (0...1) of the trailer.
    func scrub(to fraction: Double) {
        guard duration > 0 else { return }
        let f = min(max(fraction, 0), 1)
        progress = f
        currentTime = f * duration
        player.seek(to: CMTime(seconds: currentTime, preferredTimescale: 600), toleranceBefore: .zero, toleranceAfter: .zero)
    }

    /// The finger leaves the bar: playback goes on from there.
    func endScrub() {
        scrubbing = false
        if wantPlaying { player.play() }
    }

    /// The layer has a frame to show (even while paused): the picture can appear before the first play.
    func markReady() { isReady = true }

    /// Stops everything (the card left the pool).
    func stop() {
        wantPlaying = false
        player.pause()
        player.replaceCurrentItem(with: nil)
        if let timeObserver { player.removeTimeObserver(timeObserver) }
        timeObserver = nil
        if let endObserver { NotificationCenter.default.removeObserver(endObserver) }
        endObserver = nil
        observers.removeAll()
    }
}

/// The videos of the current and the next card; every other one is released.
@MainActor
final class TeaserVideoPool {
    private let token: String?
    private var videos: [String: TeaserVideo] = [:]

    init(token: String?) { self.token = token }

    func video(for teaser: Teaser) -> TeaserVideo {
        if let known = videos[teaser.id] { return known }
        let video = TeaserVideo(url: teaser.videoURL, token: token)
        videos[teaser.id] = video
        return video
    }

    /// Keeps these ids (current and next) and releases the rest.
    func keep(_ ids: Set<String>) {
        for (id, video) in videos where !ids.contains(id) {
            video.stop()
            videos[id] = nil
        }
    }

    func removeAll() { keep([]) }
}

/// The picture of a `TeaserVideo`.
struct TeaserVideoView: UIViewRepresentable {
    let video: TeaserVideo
    /// Zoomed in: the picture fills the page even when the trailer is landscape.
    var zoomed = false

    func makeUIView(context: Context) -> PlayerLayerView {
        let view = PlayerLayerView()
        view.isUserInteractionEnabled = false
        configure(view)
        return view
    }

    func updateUIView(_ view: PlayerLayerView, context: Context) { configure(view) }

    /// The layer follows the current player and mode (a view that came back from another tab may hold an old player).
    private func configure(_ view: PlayerLayerView) {
        if view.playerLayer.player !== video.player { view.playerLayer.player = video.player }
        view.playerLayer.videoGravity = (video.fillsPage || zoomed) ? .resizeAspectFill : .resizeAspect
        view.onReady = { [video] in video.markReady() }
    }

    final class PlayerLayerView: UIView {
        override static var layerClass: AnyClass { AVPlayerLayer.self }
        var playerLayer: AVPlayerLayer { layer as! AVPlayerLayer }
        /// Called when the layer has its first frame, so a buffered trailer shows at once.
        var onReady: () -> Void = {}
        private var observation: NSKeyValueObservation?

        override init(frame: CGRect) {
            super.init(frame: frame)
            observation = playerLayer.observe(\.isReadyForDisplay, options: [.initial, .new]) { [weak self] layer, _ in
                guard layer.isReadyForDisplay else { return }
                Task { @MainActor in self?.onReady() }
            }
        }

        required init?(coder: NSCoder) { fatalError("init(coder:) is not used") }
    }
}
