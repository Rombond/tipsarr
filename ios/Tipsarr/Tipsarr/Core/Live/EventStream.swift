import Foundation

/// One server-sent event: `request.updated`, `request.progress`, `media.available`,
/// `suggestions.updated`, `issue.updated`. Events are nudges; the data only identifies what changed.
struct LiveEvent: Sendable {
    var id: Int?
    var type: String
    /// Request or issue id, when the event has one.
    var itemID: String?
    var percent: Int?
    var etaSeconds: Int?
}

/// Streams `GET /api/v1/events` (SSE) and reconnects with `Last-Event-ID`.
/// The caller cancels the task to stop it (app in the background, account switched).
enum EventStream {
    /// `onEvent` runs for every event; `onConnected` runs after every (re)connection so callers refetch.
    static func run(server: URL, token: String, onConnected: @MainActor @Sendable () -> Void, onEvent: @MainActor @Sendable (LiveEvent) -> Void) async {
        let configuration = URLSessionConfiguration.default
        configuration.timeoutIntervalForRequest = 90 // the server pings every 20 s
        configuration.timeoutIntervalForResource = .infinity
        configuration.urlCache = nil
        let session = URLSession(configuration: configuration)
        var lastID: Int?
        var delay: Double = 1

        while !Task.isCancelled {
            var request = URLRequest(url: server.appending(path: "api/v1/events"))
            request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
            request.setValue("text/event-stream", forHTTPHeaderField: "Accept")
            request.setValue("ios/\(TipsarrClient.appVersion)", forHTTPHeaderField: "X-Tipsarr-App")
            if let lastID { request.setValue(String(lastID), forHTTPHeaderField: "Last-Event-ID") }
            do {
                let (bytes, response) = try await session.bytes(for: request)
                guard (response as? HTTPURLResponse)?.statusCode == 200 else { throw URLError(.badServerResponse) }
                delay = 1
                await onConnected()
                var buffer = Data()
                for try await byte in bytes {
                    buffer.append(byte)
                    // A blank line ends an event.
                    if buffer.suffix(2) == Data([0x0A, 0x0A]), let event = parse(buffer) {
                        lastID = event.id ?? lastID
                        await onEvent(event)
                        buffer.removeAll(keepingCapacity: true)
                    } else if buffer.suffix(2) == Data([0x0A, 0x0A]) {
                        buffer.removeAll(keepingCapacity: true)
                    }
                }
            } catch {
                if Task.isCancelled { return }
            }
            // Dropped or refused: wait, then try again (3 s at first, 30 s at most).
            try? await Task.sleep(for: .seconds(delay))
            delay = min(delay * 2, 30)
        }
    }

    /// Lines `id:`, `event:`, `data:`; comments (`: ping`) and the `retry:` hint give no event.
    private static func parse(_ block: Data) -> LiveEvent? {
        guard let text = String(data: block, encoding: .utf8) else { return nil }
        var id: Int?
        var type: String?
        var data = ""
        for line in text.split(separator: "\n", omittingEmptySubsequences: true) {
            if line.hasPrefix("id:") { id = Int(line.dropFirst(3).trimmingCharacters(in: .whitespaces)) }
            else if line.hasPrefix("event:") { type = line.dropFirst(6).trimmingCharacters(in: .whitespaces) }
            else if line.hasPrefix("data:") { data += line.dropFirst(5).trimmingCharacters(in: .whitespaces) }
        }
        guard let type else { return nil }
        let object = (data.data(using: .utf8)).flatMap { try? JSONSerialization.jsonObject(with: $0) as? [String: Any] } ?? [:]
        return LiveEvent(id: id, type: type, itemID: object["id"] as? String,
                         percent: object["percent"] as? Int, etaSeconds: object["etaSeconds"] as? Int)
    }
}
