import SwiftUI
import UIKit
import os

/// Where images come from: the active account's server and token.
struct ImageSource: Sendable, Equatable {
    var serverURL: URL
    var token: String

    /// A path on the Tipsarr server itself, e.g. `/api/v1/images/jellyfin/<id>?tag=<tag>`.
    func serverURL(path: String) -> URL? {
        URL(string: serverURL.absoluteString + (path.hasPrefix("/") ? path : "/" + path))
    }

    /// `path` is a TMDB path such as `/abc.jpg`.
    func tmdbURL(size: TMDBSize, path: String) -> URL? {
        let file = path.hasPrefix("/") ? path : "/" + path
        return URL(string: serverURL.absoluteString + "/api/v1/images/tmdb/\(size.rawValue)" + file)
    }
}

enum TMDBSize: String, Sendable {
    case w92, w185, w342, w500, w780, w1280
}

/// Bearer-aware image loader: memory cache in front of a 200 MB disk cache.
/// TMDB images are immutable, so cached copies are always preferred.
actor ImageLoader {
    static let shared = ImageLoader()
    private static let logger = Logger(subsystem: "com.brebond.tipsarr", category: "network")

    private let memory = NSCache<NSURL, UIImage>()
    private let session: URLSession
    private var inFlight: [URL: Task<UIImage, Error>] = [:]

    init() {
        let configuration = URLSessionConfiguration.default
        configuration.urlCache = URLCache(memoryCapacity: 20 << 20, diskCapacity: 200 << 20, directory: nil)
        configuration.requestCachePolicy = .returnCacheDataElseLoad
        configuration.timeoutIntervalForRequest = 15
        session = URLSession(configuration: configuration)
        memory.totalCostLimit = 64 << 20
    }

    func image(url: URL, token: String) async throws -> UIImage {
        if let cached = memory.object(forKey: url as NSURL) { return cached }
        if let task = inFlight[url] { return try await task.value }
        let task = Task { [session] () throws -> UIImage in
            var request = URLRequest(url: url)
            request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
            let start = ContinuousClock.now
            let (data, response) = try await session.data(for: request)
            let ms = (ContinuousClock.now - start).milliseconds
            Self.logger.notice("image \(url.lastPathComponent, privacy: .public) \((response as? HTTPURLResponse)?.statusCode ?? 0) \(ms) ms")
            #if DEBUG
            print("[network] image \(url.lastPathComponent) \((response as? HTTPURLResponse)?.statusCode ?? 0) \(ms) ms")
            #endif
            guard (response as? HTTPURLResponse)?.statusCode == 200,
                  let image = UIImage(data: data)?.preparingForDisplay() else { throw URLError(.cannotDecodeContentData) }
            return image
        }
        inFlight[url] = task
        defer { inFlight[url] = nil }
        let image = try await task.value
        memory.setObject(image, forKey: url as NSURL, cost: Int(image.size.width * image.size.height * 4))
        return image
    }
}

private struct ImageSourceKey: EnvironmentKey {
    static let defaultValue: ImageSource? = nil
}

extension EnvironmentValues {
    var imageSource: ImageSource? {
        get { self[ImageSourceKey.self] }
        set { self[ImageSourceKey.self] = newValue }
    }
}

/// Loads a TMDB image; shows `placeholder` until it arrives or when it fails.
struct RemoteImage<Placeholder: View>: View {
    enum Reference: Equatable {
        case tmdb(path: String?, size: TMDBSize)
        case server(path: String?)
    }

    let reference: Reference
    @ViewBuilder var placeholder: Placeholder

    init(path: String?, size: TMDBSize, @ViewBuilder placeholder: () -> Placeholder) {
        reference = .tmdb(path: path, size: size)
        self.placeholder = placeholder()
    }

    /// For images the server serves from its own path (Jellyfin posters of the library).
    init(serverPath: String?, @ViewBuilder placeholder: () -> Placeholder) {
        reference = .server(path: serverPath)
        self.placeholder = placeholder()
    }

    @Environment(\.imageSource) private var source
    @State private var image: UIImage?

    var body: some View {
        // The container decides the size; a fill-scaled image would otherwise report a size
        // wider than its frame and push the whole screen wider than the display.
        Color.clear
            .overlay {
                if let image {
                    Image(uiImage: image).resizable().scaledToFill().transition(.opacity)
                } else {
                    placeholder
                }
            }
            .clipped()
        .animation(.easeOut(duration: Tokens.Motion.normal), value: image)
        .task(id: TaskKey(reference: reference, source: source)) {
            guard let source, let url = url(for: source) else {
                image = nil
                return
            }
            image = try? await ImageLoader.shared.image(url: url, token: source.token)
        }
    }

    private func url(for source: ImageSource) -> URL? {
        switch reference {
        case .tmdb(let path?, let size): source.tmdbURL(size: size, path: path)
        case .server(let path?): source.serverURL(path: path)
        default: nil
        }
    }

    private struct TaskKey: Equatable {
        var reference: Reference
        var source: ImageSource?
    }
}
