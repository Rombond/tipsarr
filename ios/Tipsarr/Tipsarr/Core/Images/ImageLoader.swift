import SwiftUI
import UIKit
import os

/// Where images come from: the active account's server and token.
struct ImageSource: Sendable, Equatable {
    var serverURL: URL
    var token: String

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
    let path: String?
    let size: TMDBSize
    @ViewBuilder var placeholder: Placeholder

    @Environment(\.imageSource) private var source
    @State private var image: UIImage?

    var body: some View {
        ZStack {
            if let image {
                Image(uiImage: image).resizable().scaledToFill().transition(.opacity)
            } else {
                placeholder
            }
        }
        .animation(.easeOut(duration: Tokens.Motion.normal), value: image)
        .task(id: TaskKey(path: path, source: source)) {
            guard let path, let source, let url = source.tmdbURL(size: size, path: path) else {
                image = nil
                return
            }
            image = try? await ImageLoader.shared.image(url: url, token: source.token)
        }
    }

    private struct TaskKey: Equatable {
        var path: String?
        var source: ImageSource?
    }
}
