import Foundation

enum AppVersion {
    /// True when `current` is older than `minimum`. An empty minimum means any version.
    static func isOlder(_ current: String, than minimum: String) -> Bool {
        guard !minimum.isEmpty else { return false }
        let a = parts(current), b = parts(minimum)
        for i in 0..<max(a.count, b.count) {
            let x = i < a.count ? a[i] : 0, y = i < b.count ? b[i] : 0
            if x != y { return x < y }
        }
        return false
    }

    private static func parts(_ version: String) -> [Int] {
        version.split(separator: ".").map { Int($0.prefix { $0.isNumber }) ?? 0 }
    }
}

enum ServerAddress {
    /// "tipsarr.example.com/" → https://tipsarr.example.com. Nil when it cannot be a server URL.
    static func normalize(_ raw: String) -> URL? {
        var text = raw.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !text.isEmpty else { return nil }
        if !text.contains("://") { text = "https://" + text }
        guard var components = URLComponents(string: text),
              let scheme = components.scheme?.lowercased(), ["http", "https"].contains(scheme),
              let host = components.host, !host.isEmpty else { return nil }
        components.scheme = scheme
        components.query = nil
        components.fragment = nil
        while components.path.hasSuffix("/") { components.path.removeLast() }
        return components.url
    }
}
