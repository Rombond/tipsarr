import SwiftUI

struct IssueDetailView: View {
    let id: String
    @Environment(\.appContext) private var context
    @Environment(ToastCenter.self) private var toast
    @Environment(\.liveUpdates) private var live
    @Environment(\.dismiss) private var dismiss
    @State private var thread: IssueThread?
    @State private var error: APIError?
    @State private var reply = ""
    @State private var sending = false
    @State private var confirmDelete = false

    var body: some View {
        Group {
            if let thread { content(thread) } else if let error {
                ErrorState(error: error) { await load() }
            } else {
                ProgressView().frame(maxWidth: .infinity, maxHeight: .infinity)
            }
        }
        .background(Tokens.palette.bg)
        .navigationTitle("issue.title")
        .navigationBarTitleDisplayMode(.inline)
        .task { await load() }
        .onChange(of: live?.issuesTick) { Task { await load() } }
    }

    private func content(_ thread: IssueThread) -> some View {
        let issue = thread.issue
        return ScrollView {
            VStack(alignment: .leading, spacing: Tokens.Spacing.xl) {
                MediaLink(route: issue.route) {
                    HStack(spacing: Tokens.Spacing.md) {
                        RemoteImage(path: issue.posterPath, size: .w185) {
                            Image(systemName: "film").foregroundStyle(Tokens.palette.mutedFg)
                        }
                        .frame(width: 64, height: 96)
                        .background(Tokens.palette.muted)
                        .clipShape(.rect(cornerRadius: Tokens.Radius.sm))
                        .accessibilityHidden(true)
                        VStack(alignment: .leading, spacing: Tokens.Spacing.xs) {
                            Text(verbatim: issue.title).font(.title3.weight(.bold)).foregroundStyle(Tokens.palette.fg)
                            HStack(spacing: Tokens.Spacing.sm) {
                                Text(issue.kind.title).font(.caption.weight(.semibold))
                                    .padding(.horizontal, Tokens.Spacing.sm).padding(.vertical, 2)
                                    .background(Tokens.palette.muted, in: .capsule)
                                if let scope = IssueScope.text(season: issue.season, episode: issue.episode) {
                                    Text(verbatim: scope).font(.caption).foregroundStyle(Tokens.palette.mutedFg)
                                }
                            }
                            Text(issue.status == .open ? "issue.open" : "issue.resolved")
                                .font(.subheadline.weight(.semibold))
                                .foregroundStyle(issue.status == .open ? Tokens.Status.requested : Tokens.Status.available)
                            if issue.status == .resolved, let by = issue.resolvedBy {
                                Text(verbatim: L10n.string("issue.resolved_by", by)).font(.footnote).foregroundStyle(Tokens.palette.mutedFg)
                            }
                        }
                        Spacer(minLength: 0)
                        Image(systemName: "chevron.right").font(.footnote).foregroundStyle(Tokens.palette.mutedFg)
                    }
                }
                .buttonStyle(.plain)
                VStack(alignment: .leading, spacing: Tokens.Spacing.md) {
                    Text("issue.thread").font(.title3.weight(.bold)).accessibilityAddTraits(.isHeader)
                    if thread.comments.isEmpty {
                        Text("m.admin.no_comments").font(.subheadline).foregroundStyle(Tokens.palette.mutedFg)
                    }
                    ForEach(thread.comments) { comment in CommentRow(comment: comment) }
                }
                replyBox
                actions(issue)
            }
            .padding(Tokens.Spacing.lg)
            .frame(maxWidth: 720)
            .frame(maxWidth: .infinity)
        }
        .scrollDismissesKeyboard(.interactively)
    }

    private var replyBox: some View {
        VStack(alignment: .leading, spacing: Tokens.Spacing.sm) {
            TextField("issue.reply_placeholder", text: $reply, axis: .vertical)
                .lineLimit(2...6)
                .padding(Tokens.Spacing.md)
                .background(Tokens.palette.muted, in: .rect(cornerRadius: Tokens.Radius.md))
            Button { Task { await send() } } label: {
                Label { Text(sending ? "common.saving" : "issue.comment") } icon: { Image(systemName: "paperplane") }
            }
            .buttonStyle(.tipsarr(.primary, fullWidth: true))
            .disabled(sending || reply.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
        }
    }

    private func actions(_ issue: IssueRecord) -> some View {
        VStack(spacing: Tokens.Spacing.sm) {
            if context?.profile.isAdmin == true {
                Button { Task { await toggle(issue) } } label: {
                    Label { Text(issue.status == .open ? "issue.resolve" : "issue.reopen") }
                        icon: { Image(systemName: issue.status == .open ? "checkmark.circle" : "arrow.uturn.backward.circle") }
                }
                .buttonStyle(.tipsarr(.secondary, fullWidth: true))
                Button { confirmDelete = true } label: { Text("common.delete") }
                    .buttonStyle(.tipsarr(.ghost, fullWidth: true))
                    .foregroundStyle(Tokens.palette.destructive)
                    .confirmationDialog(Text("issue.confirm_delete"), isPresented: $confirmDelete, titleVisibility: .visible) {
                        Button("common.delete", role: .destructive) { Task { await delete() } }
                        Button("common.cancel", role: .cancel) {}
                    }
            }
        }
    }

    // MARK: Data

    private func load() async {
        guard let context else { return }
        do {
            thread = try await context.api.issue(id: id)
            error = nil
        } catch {
            if thread == nil { self.error = APIError.from(error) }
        }
    }

    private func send() async {
        guard let context else { return }
        sending = true
        defer { sending = false }
        do {
            thread = try await context.api.comment(on: id, message: reply.trimmingCharacters(in: .whitespacesAndNewlines))
            reply = ""
        } catch {
            toast.show(APIError.from(error).localizedMessage, kind: .error)
        }
    }

    private func toggle(_ issue: IssueRecord) async {
        guard let context else { return }
        do {
            thread = issue.status == .open ? try await context.api.resolveIssue(id: id) : try await context.api.reopenIssue(id: id)
        } catch {
            toast.show(APIError.from(error).localizedMessage, kind: .error)
        }
    }

    private func delete() async {
        guard let context else { return }
        do {
            try await context.api.deleteIssue(id: id)
            dismiss()
        } catch {
            toast.show(APIError.from(error).localizedMessage, kind: .error)
        }
    }
}

private struct CommentRow: View {
    let comment: IssueComment

    var body: some View {
        HStack(alignment: .top, spacing: Tokens.Spacing.md) {
            AvatarView(userID: comment.userID, name: comment.userName, size: 36)
            VStack(alignment: .leading, spacing: Tokens.Spacing.xs) {
                HStack {
                    Text(verbatim: comment.userName).font(.subheadline.weight(.semibold))
                    Text(comment.createdAt, format: .relative(presentation: .named)).font(.caption).foregroundStyle(Tokens.palette.mutedFg)
                }
                Text(verbatim: comment.message).font(.callout)
            }
            Spacer(minLength: 0)
        }
        .accessibilityElement(children: .combine)
    }
}
