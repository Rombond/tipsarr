import Foundation
import OpenAPIRuntime

/// The calls the launch flow needs. Throws `APIError` only.
struct TipsarrAPI: Sendable {
    let serverURL: URL
    let token: String?

    private var client: Client { TipsarrClient.make(server: serverURL, token: token) }

    func status() async throws -> ServerStatus {
        do {
            guard case .ok(let ok) = try await client.status() else { throw APIError.unexpected }
            let body = try ok.body.json
            return ServerStatus(
                version: body.version,
                apiVersion: Int(body.apiVersion),
                minAppVersion: body.minAppVersion,
                userFolderChoice: body.userFolderChoice,
                defaultLanguage: body.defaultLanguage,
                pushAvailable: body.features.push
            )
        } catch let error as DecodingError {
            // A reachable server whose /status is not ours is "not a Tipsarr server".
            _ = error
            throw APIError.unexpected
        } catch {
            throw APIError.from(error)
        }
    }

    func signIn(username: String, password: String, deviceName: String) async throws -> SignInResult {
        do {
            let input = Components.Schemas.TokenInputBody(
                appVersion: TipsarrClient.appVersion,
                deviceName: deviceName,
                password: password,
                platform: .ios,
                username: username
            )
            guard case .ok(let ok) = try await client.loginToken(body: .json(input)) else { throw APIError.unexpected }
            let body = try ok.body.json
            return SignInResult(token: body.token, profile: Profile(body.user))
        } catch {
            throw APIError.from(error)
        }
    }

    func me() async throws -> Profile {
        do {
            guard case .ok(let ok) = try await client.me() else { throw APIError.unexpected }
            return Profile(try ok.body.json)
        } catch {
            throw APIError.from(error)
        }
    }

    /// Best effort: the account is removed from the phone even when this fails.
    func logout() async {
        _ = try? await client.logout()
    }
}

private extension Profile {
    init(_ user: Components.Schemas.User) {
        self.init(id: user.id, name: user.name, role: user.role, region: user.region,
                  language: user.language, ratingSource: user.ratingSource)
    }
}
