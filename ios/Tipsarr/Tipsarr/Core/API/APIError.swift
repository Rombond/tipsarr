import Foundation

/// Every failure the API layer can surface. Non-2xx responses become `.http`
/// (see `ErrorMiddleware`); transport failures become `.unreachable`.
enum APIError: Error, Sendable, Equatable {
    case unreachable
    case http(status: Int, code: String?, detail: String?)
    case unexpected

    var status: Int? {
        if case .http(let status, _, _) = self { status } else { nil }
    }

    var code: String? {
        if case .http(_, let code, _) = self { code } else { nil }
    }

    /// True for an authenticated call that the server refused with 401.
    var isUnauthorized: Bool { status == 401 }

    /// Localized text for the error `code`, then its HTTP status, then the server `detail`.
    var localizedMessage: String {
        switch self {
        case .unreachable:
            return Self.lookup("m.connect.unreachable") ?? ""
        case .unexpected:
            return Self.lookup("error.status_500") ?? ""
        case .http(let status, let code, let detail):
            if let code, let text = Self.lookup("error.\(code)") { return text }
            if let text = Self.lookup("error.status_\(status)") { return text }
            return detail ?? Self.lookup("error.status_500") ?? ""
        }
    }

    /// Wraps any error thrown by the transport or the generated client.
    static func from(_ error: Error) -> APIError {
        if let error = error as? APIError { return error }
        return isNetwork(error) ? .unreachable : .unexpected
    }

    /// Network-level failures that are worth retrying on a GET.
    static func isNetwork(_ error: Error) -> Bool {
        let underlying: Error = (error as? ClientErrorProviding)?.underlying ?? error
        guard let urlError = underlying as? URLError else { return false }
        return urlError.code != .cancelled
    }

    private static func lookup(_ key: String) -> String? {
        let text = AppLanguage.bundle.localizedString(forKey: key, value: nil, table: nil)
        return text == key ? nil : text
    }
}

/// Lets `APIError` unwrap `ClientError` without importing the runtime here.
protocol ClientErrorProviding { var underlying: Error { get } }
