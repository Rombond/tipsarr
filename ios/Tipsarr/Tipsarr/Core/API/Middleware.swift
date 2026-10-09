import Foundation
import HTTPTypes
import OpenAPIRuntime

extension ClientError: ClientErrorProviding {
    var underlying: Error { underlyingError }
}

/// Adds the headers of the behaviour spec to every call.
struct HeadersMiddleware: ClientMiddleware {
    let token: String?
    let language: String
    let appVersion: String

    private static let languageName = HTTPField.Name("X-Tipsarr-Language")!
    private static let appName = HTTPField.Name("X-Tipsarr-App")!

    func intercept(
        _ request: HTTPRequest,
        body: HTTPBody?,
        baseURL: URL,
        operationID: String,
        next: @concurrent @Sendable (HTTPRequest, HTTPBody?, URL) async throws -> (HTTPResponse, HTTPBody?)
    ) async throws -> (HTTPResponse, HTTPBody?) {
        var request = request
        if let token { request.headerFields[.authorization] = "Bearer \(token)" }
        request.headerFields[Self.languageName] = language
        request.headerFields[Self.appName] = "ios/\(appVersion)"
        return try await next(request, body, baseURL)
    }
}

/// GET only: retry twice (1 s, 3 s) on network errors and 502/503. Writes are never retried.
struct RetryMiddleware: ClientMiddleware {
    private let delays: [Duration] = [.seconds(1), .seconds(3)]

    func intercept(
        _ request: HTTPRequest,
        body: HTTPBody?,
        baseURL: URL,
        operationID: String,
        next: @concurrent @Sendable (HTTPRequest, HTTPBody?, URL) async throws -> (HTTPResponse, HTTPBody?)
    ) async throws -> (HTTPResponse, HTTPBody?) {
        guard request.method == .get else { return try await next(request, body, baseURL) }
        var attempt = 0
        while true {
            do {
                let result = try await next(request, body, baseURL)
                let code = result.0.status.code
                if (code == 502 || code == 503), attempt < delays.count {
                    try await Task.sleep(for: delays[attempt])
                    attempt += 1
                    continue
                }
                return result
            } catch where APIError.isNetwork(error) && attempt < delays.count {
                try await Task.sleep(for: delays[attempt])
                attempt += 1
            }
        }
    }
}

/// Turns every non-2xx response into an `APIError.http` so call sites only handle success.
struct ErrorMiddleware: ClientMiddleware {
    func intercept(
        _ request: HTTPRequest,
        body: HTTPBody?,
        baseURL: URL,
        operationID: String,
        next: @concurrent @Sendable (HTTPRequest, HTTPBody?, URL) async throws -> (HTTPResponse, HTTPBody?)
    ) async throws -> (HTTPResponse, HTTPBody?) {
        let (response, responseBody) = try await next(request, body, baseURL)
        guard !(200..<300).contains(response.status.code) else { return (response, responseBody) }
        var data = Data()
        if let responseBody { data = Data(try await ArraySlice(collecting: responseBody, upTo: 1 << 20)) }
        let (code, detail) = Self.parse(data)
        throw APIError.http(status: response.status.code, code: code, detail: detail)
    }

    /// Server errors carry `errors[0].location == "code"` with the stable code in `value`.
    private static func parse(_ data: Data) -> (code: String?, detail: String?) {
        guard let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any] else { return (nil, nil) }
        let errors = json["errors"] as? [[String: Any]] ?? []
        let code = errors.first { $0["location"] as? String == "code" }?["value"] as? String
        return (code, json["detail"] as? String)
    }
}
