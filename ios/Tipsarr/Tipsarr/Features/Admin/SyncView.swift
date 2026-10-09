import SwiftUI

/// Background jobs of the server (library sync, history sync, box office, Radarr/Sonarr import, playback).
struct SyncView: View {
    @Environment(\.appContext) private var context
    @Environment(\.liveUpdates) private var live
    @Environment(ToastCenter.self) private var toast
    @State private var status: SyncStatus?
    @State private var error: APIError?

    var body: some View {
        Group {
            if let status { content(status) } else if let error {
                ErrorState(error: error) { await load() }
            } else {
                ProgressView().frame(maxWidth: .infinity, maxHeight: .infinity)
            }
        }
        .background(Tokens.palette.bg)
        .navigationTitle("m.admin.sync")
        .navigationBarTitleDisplayMode(.inline)
        .task { await load() }
        .refreshable { await load() }
        .onChange(of: live?.syncTick) { Task { await load() } }
        // Progress is pushed by the server; this only covers a dropped stream while a job runs.
        .task(id: status?.jobs.contains { $0.running }) {
            while status?.jobs.contains(where: { $0.running }) == true {
                try? await Task.sleep(for: .seconds(5))
                if Task.isCancelled { return }
                await load()
            }
        }
    }

    private func content(_ status: SyncStatus) -> some View {
        List {
            Section {
                Text(verbatim: L10n.string("m.sync.counts", String(status.movies), String(status.shows)))
                    .font(.subheadline).foregroundStyle(Tokens.palette.mutedFg)
                    .listRowBackground(Color.clear)
                if !status.canSync { Banner(kind: .warning, title: "m.sync.need_key").listRowBackground(Color.clear) }
            }
            ForEach(status.jobs) { job in
                JobCard(job: job, canRun: status.canSync || !["library-sync", "history-sync"].contains(job.name)) {
                    await run(job)
                }
                .listRowBackground(Color.clear)
                .listRowSeparator(.hidden)
            }
        }
        .listStyle(.plain)
    }

    private func load() async {
        guard let context else { return }
        do {
            status = try await context.api.syncStatus()
            error = nil
        } catch {
            if status == nil { self.error = APIError.from(error) }
        }
    }

    private func run(_ job: JobStatus) async {
        guard let context else { return }
        do {
            try await context.api.runJob(job.name)
            toast.show(L10n.string("m.sync.started"))
            await load()
        } catch {
            toast.show(APIError.from(error).localizedMessage, kind: .error)
        }
    }
}

private struct JobCard: View {
    let job: JobStatus
    let canRun: Bool
    let run: () async -> Void
    @State private var starting = false

    var body: some View {
        VStack(alignment: .leading, spacing: Tokens.Spacing.sm) {
            HStack(alignment: .firstTextBaseline) {
                Text(LText(stringLiteral: "m.job.\(job.name)")).font(.headline)
                Spacer()
                if job.running {
                    HStack(spacing: Tokens.Spacing.xs) { ProgressView().controlSize(.small); Text("m.sync.running").font(.footnote) }
                        .foregroundStyle(Tokens.Status.downloading)
                }
            }
            Text(LText(stringLiteral: "m.job.\(job.name).desc")).font(.footnote).foregroundStyle(Tokens.palette.mutedFg)
            Group {
                if let finished = job.lastFinished {
                    Text(verbatim: L10n.string("m.sync.last", finished.formatted(.relative(presentation: .named))))
                } else {
                    Text("m.sync.never")
                }
            }
            .font(.footnote)
            .foregroundStyle(Tokens.palette.mutedFg)
            if !job.message.isEmpty {
                Text(verbatim: job.message).font(.footnote).foregroundStyle(job.status == "error" ? Tokens.palette.destructive : Tokens.palette.mutedFg)
            }
            HStack {
                if job.everySeconds > 0 {
                    Text(verbatim: L10n.string("m.sync.every", Duration.seconds(job.everySeconds).formatted(.units(allowed: [.hours, .minutes], width: .wide))))
                        .font(.caption).foregroundStyle(Tokens.palette.mutedFg)
                }
                Spacer()
                Button { Task { starting = true; await run(); starting = false } } label: {
                    Label { Text("m.sync.run") } icon: { Image(systemName: "play.fill") }
                }
                .buttonStyle(.tipsarr(.secondary, compact: true))
                .disabled(job.running || starting || !canRun)
            }
        }
        .padding(Tokens.Spacing.md)
        .background(Tokens.palette.card, in: .rect(cornerRadius: Tokens.Radius.md))
        .overlay { RoundedRectangle(cornerRadius: Tokens.Radius.md).strokeBorder(Tokens.palette.border) }
    }
}
