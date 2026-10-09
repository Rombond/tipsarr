import Foundation
import OpenAPIRuntime

extension TipsarrAPI {
    func updatePreferences(language: String? = nil, ratingSource: RatingSource? = nil, region: String? = nil) async throws -> Profile {
        try await load { client in
            let body = Components.Schemas.PrefsBody(
                language: language,
                ratingSource: ratingSource.flatMap { .init(rawValue: $0.rawValue) },
                region: region
            )
            guard case .ok(let ok) = try await client.updateMe(body: .json(body)) else { throw APIError.unexpected }
            return Profile(try ok.body.json)
        }
    }

    func sessions() async throws -> [DeviceSession] {
        try await load { client in
            guard case .ok(let ok) = try await client.listSessions() else { throw APIError.unexpected }
            return try ok.body.json.map { item in
                DeviceSession(
                    id: item.id,
                    platform: .init(rawValue: item.platform.rawValue) ?? .web,
                    deviceName: item.deviceName,
                    appVersion: item.appVersion,
                    lastSeen: Date(timeIntervalSince1970: TimeInterval(item.lastSeenAt)),
                    current: item.current
                )
            }
        }
    }

    func revokeSession(id: String) async throws {
        try await load { client in _ = try await client.revokeSession(path: .init(id: id)) }
    }

    func revokeOtherSessions() async throws {
        try await load { client in _ = try await client.revokeOtherSessions() }
    }

    /// Avatar upload is a raw multipart route, outside the generated client.
    func uploadAvatar(jpeg: Data) async throws {
        let boundary = "tipsarr-\(UUID().uuidString)"
        var request = URLRequest(url: serverURL.appending(path: "api/v1/me/avatar"))
        request.httpMethod = "POST"
        request.timeoutInterval = 30
        request.setValue("multipart/form-data; boundary=\(boundary)", forHTTPHeaderField: "Content-Type")
        if let token { request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization") }
        var body = Data()
        body.append(Data("--\(boundary)\r\nContent-Disposition: form-data; name=\"file\"; filename=\"avatar.jpg\"\r\nContent-Type: image/jpeg\r\n\r\n".utf8))
        body.append(jpeg)
        body.append(Data("\r\n--\(boundary)--\r\n".utf8))
        try await send(request, body: body)
    }

    func deleteAvatar() async throws {
        var request = URLRequest(url: serverURL.appending(path: "api/v1/me/avatar"))
        request.httpMethod = "DELETE"
        if let token { request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization") }
        try await send(request, body: nil)
    }

    private func send(_ request: URLRequest, body: Data?) async throws {
        do {
            let (data, response) = try await URLSession.shared.upload(for: request, from: body ?? Data())
            guard let http = response as? HTTPURLResponse else { throw APIError.unexpected }
            guard (200..<300).contains(http.statusCode) else {
                let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any]
                let errors = json?["errors"] as? [[String: Any]] ?? []
                let code = errors.first { $0["location"] as? String == "code" }?["value"] as? String
                throw APIError.http(status: http.statusCode, code: code, detail: json?["detail"] as? String)
            }
        } catch {
            throw APIError.from(error)
        }
    }
}
