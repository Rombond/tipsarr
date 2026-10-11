import SwiftUI
import WebKit

/// A YouTube trailer in an embedded player (the only way YouTube allows). Not touchable: the page above handles taps and swipes.
struct TeaserPlayer: UIViewRepresentable {
    let videoID: String
    var muted: Bool
    var playing: Bool
    /// YouTube error code: 100 (gone), 101 and 150 (the owner forbids embedding) and so on.
    var onError: (Int) -> Void = { _ in }

    func makeCoordinator() -> Coordinator { Coordinator(onError: onError) }

    func makeUIView(context: Context) -> WKWebView {
        let config = WKWebViewConfiguration()
        config.allowsInlineMediaPlayback = true
        // Sound starts by itself: the person opened the Teasers tab to watch trailers.
        config.mediaTypesRequiringUserActionForPlayback = []
        config.userContentController.add(context.coordinator, name: "yt")
        let view = WKWebView(frame: .zero, configuration: config)
        view.isOpaque = false
        view.backgroundColor = .black
        view.scrollView.isScrollEnabled = false
        view.isUserInteractionEnabled = false
        context.coordinator.webView = view
        context.coordinator.last = State(muted: muted, playing: playing)
        view.loadHTMLString(Self.page(videoID: videoID, muted: muted, playing: playing), baseURL: Self.origin)
        return view
    }

    func updateUIView(_ view: WKWebView, context: Context) {
        context.coordinator.onError = onError
        let state = State(muted: muted, playing: playing)
        guard context.coordinator.last != state else { return }
        context.coordinator.last = state
        view.evaluateJavaScript("setWant(\(muted), \(playing))")
    }

    static func dismantleUIView(_ view: WKWebView, coordinator: Coordinator) {
        view.configuration.userContentController.removeScriptMessageHandler(forName: "yt")
        view.stopLoading()
        view.loadHTMLString("", baseURL: nil)
    }

    /// YouTube wants a real https page as the embedding origin; the page is loaded from a string, so this stands in for it.
    private static let origin = URL(string: "https://tipsarr.app")!

    struct State: Equatable { var muted: Bool; var playing: Bool }

    final class Coordinator: NSObject, WKScriptMessageHandler {
        var onError: (Int) -> Void
        var last = State(muted: false, playing: true)
        weak var webView: WKWebView?

        init(onError: @escaping (Int) -> Void) { self.onError = onError }

        func userContentController(_ controller: WKUserContentController, didReceive message: WKScriptMessage) {
            guard let body = message.body as? [String: Any], body["e"] as? String == "error", let code = body["v"] as? Int else { return }
            onError(code)
        }
    }

    /// The IFrame API page: a 16:9 player centred on black, looping, with the controls hidden.
    private static func page(videoID: String, muted: Bool, playing: Bool) -> String {
        let safeID = videoID.filter { $0.isLetter || $0.isNumber || $0 == "-" || $0 == "_" }
        return """
        <!doctype html><html><head>
        <meta name="viewport" content="width=device-width, initial-scale=1, maximum-scale=1, user-scalable=no">
        <style>
        html, body { margin: 0; height: 100%; background: #000; overflow: hidden; }
        #wrap { position: absolute; left: 0; right: 0; top: 50%; transform: translateY(-50%); aspect-ratio: 16 / 9; }
        #p { width: 100%; height: 100%; }
        </style></head><body>
        <div id="wrap"><div id="p"></div></div>
        <script>
        var player = null, want = { muted: \(muted), playing: \(playing) };
        function send(o) { window.webkit.messageHandlers.yt.postMessage(o); }
        function apply() {
          if (!player || !player.playVideo) return;
          if (want.muted) player.mute(); else player.unMute();
          if (want.playing) player.playVideo(); else player.pauseVideo();
        }
        function setWant(m, p) { want.muted = m; want.playing = p; apply(); }
        function onYouTubeIframeAPIReady() {
          player = new YT.Player('p', {
            videoId: '\(safeID)',
            playerVars: { autoplay: 1, controls: 0, playsinline: 1, rel: 0, modestbranding: 1, fs: 0, disablekb: 1, iv_load_policy: 3, enablejsapi: 1, origin: '\(Self.origin.absoluteString)', widget_referrer: '\(Self.origin.absoluteString)' },
            events: {
              onReady: function () { apply(); send({ e: 'ready' }); },
              onStateChange: function (ev) { if (ev.data === 0) { player.seekTo(0); player.playVideo(); } send({ e: 'state', v: ev.data }); },
              onError: function (ev) { send({ e: 'error', v: ev.data }); }
            }
          });
        }
        </script>
        <script src="https://www.youtube.com/iframe_api"></script>
        </body></html>
        """
    }
}
