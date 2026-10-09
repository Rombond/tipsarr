import Foundation
import SwiftUI

struct IssueRecord: Sendable, Identifiable, Hashable {
    enum Status: String, Sendable { case open, resolved }

    var id: String
    var type: MediaType
    var tmdbId: Int
    var title: String
    var posterPath: String?
    var kind: IssueKind
    var status: Status
    var season: Int?
    var episode: Int?
    var createdBy: String
    var createdAt: Date
    var resolvedBy: String?
    var commentCount: Int

    var route: MediaRoute { MediaRoute(type: type, tmdbId: tmdbId, title: title) }
}

struct IssueComment: Sendable, Identifiable, Hashable {
    var id: String
    var message: String
    var userID: String
    var userName: String
    var createdAt: Date
}

struct IssueThread: Sendable {
    var issue: IssueRecord
    var comments: [IssueComment]
}

struct IssuePage: Sendable {
    var items: [IssueRecord]
    var total: Int
}

enum IssueFilter: String, CaseIterable, Sendable {
    case open, resolved, all

    var title: LText {
        switch self {
        case .open: "issue.filter.open"
        case .resolved: "issue.filter.resolved"
        case .all: "issue.filter.all"
        }
    }
}
