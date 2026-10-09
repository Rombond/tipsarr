import SwiftUI

struct IssuesView: View {
    @Environment(\.appContext) private var context
    @Environment(\.liveUpdates) private var live
    @State private var model: IssuesModel?
    @State private var filter: IssueFilter = .open

    var body: some View {
        Group {
            if let model { content(model) } else { ProgressView().frame(maxWidth: .infinity, maxHeight: .infinity) }
        }
        .background(Tokens.palette.bg)
        .navigationTitle("issue.title")
        .navigationBarTitleDisplayMode(.inline)
        .task { if model == nil, let context { model = IssuesModel(api: context.api) } }
    }

    private func content(_ model: IssuesModel) -> some View {
        List {
            filters(model)
            switch model.phase {
            case .idle, .loading:
                ForEach(0..<5, id: \.self) { _ in
                    HStack(alignment: .top, spacing: Tokens.Spacing.md) {
                        RoundedRectangle(cornerRadius: Tokens.Radius.sm).fill(Tokens.palette.border).frame(width: 48, height: 72)
                        VStack(alignment: .leading, spacing: Tokens.Spacing.sm) {
                            SkeletonBlock(height: 16).frame(width: 160)
                            SkeletonBlock(height: 12).frame(width: 110)
                        }
                    }
                    .skeleton()
                    .listRowBackground(Color.clear)
                }
            case .failed(let error):
                stateRow { ErrorState(error: error) { await model.load(filter) } }
            case .loaded where model.items.isEmpty:
                stateRow { StateView(symbol: "checkmark.seal", title: "issue.empty") }
            case .loaded:
                ForEach(model.items) { issue in
                    NavigationLink(value: ProfileRoute.issue(issue.id)) { IssueRow(issue: issue) }
                        .listRowBackground(Color.clear)
                        .task { await model.loadMore(after: issue) }
                }
            }
        }
        .listStyle(.plain)
        .readableColumn()
        .task(id: filter) { await model.load(filter) }
        .refreshable { await model.load(filter) }
        .onChange(of: live?.issuesTick) { Task { await model.load(filter) } }
    }

    private func filters(_ model: IssuesModel) -> some View {
        ScrollView(.horizontal, showsIndicators: false) {
            HStack(spacing: Tokens.Spacing.sm) {
                ForEach(IssueFilter.allCases, id: \.self) { item in
                    Chip(title: item.title, isSelected: filter == item, count: item == .open ? model.openCount : nil) { filter = item }
                }
            }
            .padding(.horizontal, Tokens.Spacing.lg)
        }
        .listRowInsets(.init())
        .listRowBackground(Color.clear)
        .listRowSeparator(.hidden)
    }

    private func stateRow<S: View>(@ViewBuilder _ state: () -> S) -> some View {
        state()
            .frame(minHeight: 360)
            .listRowInsets(.init())
            .listRowBackground(Color.clear)
            .listRowSeparator(.hidden)
    }
}

struct IssueRow: View {
    let issue: IssueRecord

    var body: some View {
        HStack(alignment: .top, spacing: Tokens.Spacing.md) {
            RemoteImage(path: issue.posterPath, size: .w92) {
                Image(systemName: "film").foregroundStyle(Tokens.palette.mutedFg)
            }
            .frame(width: 48, height: 72)
            .background(Tokens.palette.muted)
            .clipShape(.rect(cornerRadius: Tokens.Radius.sm))
            .accessibilityHidden(true)
            VStack(alignment: .leading, spacing: Tokens.Spacing.xs) {
                Text(verbatim: issue.title).font(.body.weight(.semibold)).lineLimit(2)
                HStack(spacing: Tokens.Spacing.sm) {
                    Text(issue.kind.title).font(.caption.weight(.semibold))
                        .padding(.horizontal, Tokens.Spacing.sm).padding(.vertical, 2)
                        .background(Tokens.palette.muted, in: .capsule)
                    if let scope = IssueScope.text(season: issue.season, episode: issue.episode) {
                        Text(verbatim: scope).font(.caption).foregroundStyle(Tokens.palette.mutedFg)
                    }
                }
                Text(verbatim: L10n.string("m.admin.reported_by", issue.createdBy) + " · " + issue.createdAt.formatted(.relative(presentation: .named)))
                    .font(.caption).foregroundStyle(Tokens.palette.mutedFg)
            }
            Spacer(minLength: 0)
            VStack(alignment: .trailing, spacing: Tokens.Spacing.xs) {
                Text(issue.status == .open ? "issue.open" : "issue.resolved")
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(issue.status == .open ? Tokens.Status.requested : Tokens.Status.available)
                if issue.commentCount > 0 {
                    Label { Text(verbatim: String(issue.commentCount)) } icon: { Image(systemName: "bubble.left") }
                        .font(.caption).foregroundStyle(Tokens.palette.mutedFg)
                }
            }
        }
        .padding(.vertical, Tokens.Spacing.xs)
        .accessibilityElement(children: .combine)
    }
}

enum IssueScope {
    /// "S2 · E4", "S2" or nil for the whole title.
    static func text(season: Int?, episode: Int?) -> String? {
        switch (season, episode) {
        case (let s?, let e?): L10n.string("issue.s_e", String(s), String(e))
        case (let s?, nil): "S\(s)"
        default: nil
        }
    }
}
