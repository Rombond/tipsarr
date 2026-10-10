import SwiftUI

struct ReportIssueSheet: View {
    let title: String
    var onSend: (IssueKind, String) async throws -> Void

    @Environment(\.dismiss) private var dismiss
    @State private var kind: IssueKind = .video
    @State private var message = ""
    @State private var sending = false
    @State private var errorText: String?
    @FocusState private var focused: Bool

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: Tokens.Spacing.lg) {
                Text(verbatim: L10n.string("issue.report_title", title)).font(.title3.weight(.semibold))
                Text("issue.report_desc").font(.subheadline).foregroundStyle(Tokens.palette.mutedFg)
                ScrollView(.horizontal, showsIndicators: false) {
                    HStack(spacing: Tokens.Spacing.sm) {
                        ForEach(IssueKind.allCases, id: \.self) { item in
                            Chip(title: item.title, isSelected: kind == item) { kind = item }
                        }
                    }
                }
                VStack(alignment: .leading, spacing: Tokens.Spacing.xs) {
                    Text("issue.describe").font(.footnote.weight(.medium)).foregroundStyle(Tokens.palette.mutedFg)
                    TextEditor(text: $message)
                        .focused($focused)
                        .scrollContentBackground(.hidden)
                        .padding(Tokens.Spacing.sm)
                        .frame(minHeight: 140)
                        .background(Tokens.palette.muted, in: .rect(cornerRadius: Tokens.Radius.md))
                        .overlay(alignment: .topLeading) {
                            if message.isEmpty {
                                Text("issue.describe_placeholder")
                                    .foregroundStyle(Tokens.palette.mutedFg)
                                    .padding(Tokens.Spacing.md)
                                    .padding(.top, 2)
                                    .allowsHitTesting(false)
                            }
                        }
                }
                if let errorText { Banner(kind: .error, title: LText.verbatim(errorText)) }
                VStack(spacing: Tokens.Spacing.sm) {
                    Button { Task { await send() } } label: {
                        Label {
                            Text(choose(sending, "common.saving", "m.issue.send"))
                        } icon: { Image(systemName: "flag") }
                    }
                    .buttonStyle(.tipsarr(.primary, fullWidth: true))
                    .disabled(sending || message.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                    Button("common.cancel") { dismiss() }
                        .buttonStyle(.tipsarr(.tonal, fullWidth: true))
                }
            }
            .padding(Tokens.Spacing.xl)
            .padding(.top, Tokens.Spacing.md)
        }
        .scrollDismissesKeyboard(.interactively)
    }

    private func send() async {
        sending = true
        errorText = nil
        defer { sending = false }
        do {
            try await onSend(kind, message.trimmingCharacters(in: .whitespacesAndNewlines))
        } catch {
            errorText = APIError.from(error).localizedMessage
        }
    }
}
