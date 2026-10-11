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
    /// The file could not be loaded or played: the page falls back to the YouTube embed.
    private(set) var failed = false
    /// Landscape trailers are shown whole on the blurred backdrop; a portrait one fills the page.
    private(set) var fillsPage = false

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
                guard let self, let duration = self.player.currentItem?.duration.seconds, duration.isFinite, duration > 0 else { return }
                self.progress = min(max(time.seconds / duration, 0), 1)
            }
        }
        endObserver = NotificationCenter.default.addObserver(forName: .AVPlayerItemDidPlayToEndTime, object: item, queue: .main) { [weak self] _ in
            Task { @MainActor in
                guard let self else { return }
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

    func video(for teaser: Teaser) -> TeaserVideo? {
        guard let url = teaser.videoURL else { return nil }
        if let known = videos[teaser.id] { return known }
        let video = TeaserVideo(url: url, token: token)
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

    func makeUIView(context: Context) -> PlayerLayerView {
        let view = PlayerLayerView()
        view.playerLayer.player = video.player
        view.playerLayer.videoGravity = video.fillsPage ? .resizeAspectFill : .resizeAspect
        view.isUserInteractionEnabled = false
        return view
    }

    func updateUIView(_ view: PlayerLayerView, context: Context) {
        view.playerLayer.videoGravity = video.fillsPage ? .resizeAspectFill : .resizeAspect
    }

    final class PlayerLayerView: UIView {
        override static var layerClass: AnyClass { AVPlayerLayer.self }
        var playerLayer: AVPlayerLayer { layer as! AVPlayerLayer }
    }
}
