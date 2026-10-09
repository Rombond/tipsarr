import Foundation
import OpenAPIRuntime
import OpenAPIURLSession

/// Builds the generated `Client` for one server (and one account's token).
enum TipsarrClient {
    static let appVersion = Bundle.main.object(forInfoDictionaryKey: "CFBundleShortVersionString") as? String ?? "0"

    /// 15 s timeouts as in the behaviour spec; no URL cache (nothing is cached).
    private static let session: URLSession = {
        let configuration = URLSessionConfiguration.default
        configuration.timeoutIntervalForRequest = 15
        configuration.timeoutIntervalForResource = 60
        configuration.urlCache = nil
        return URLSession(configuration: configuration)
    }()

    static func make(server: URL, token: String? = nil, language: String = currentLanguage) -> Client {
        Client(
            serverURL: server.appending(path: "api/v1"),
            transport: URLSessionTransport(configuration: .init(session: session)),
            middlewares: [
                TimingMiddleware(),
                HeadersMiddleware(token: token, language: language, appVersion: appVersion),
                ErrorMiddleware(),
                RetryMiddleware(),
            ]
        )
    }

    /// e.g. `fr-FR`; the profile language, when set, wins on the server.
    static var currentLanguage: String {
        (Locale.preferredLanguages.first ?? "en").replacingOccurrences(of: "_", with: "-")
    }
}
