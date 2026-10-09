import SwiftUI

/// Movie: quality profile (+ folder). Show: seasons, quality profile (+ folder).
struct RequestSheet: View {
    let model: MediaDetailModel
    var onDone: (RequestRecord) -> Void

    @Environment(\.dismiss) private var dismiss
    @State private var selectedSeasons: Set<Int> = []
    @State private var profileID: Int?
    @State private var folder: String?
    @State private var errorText: String?

    private var detail: MediaDetail? { model.detail }
    private var isTV: Bool { model.route.type == .tv }
    private var selectable: [SeasonInfo] { model.regularSeasons.filter { !model.coveredSeasons.contains($0.number) } }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: Tokens.Spacing.xl) {
                header
                if isTV { seasonPicker }
                if let options = model.options { optionRows(options) }
                Text(choose(model.isAdmin, "m.request.admin_note", "m.request.user_note"))
                    .font(.footnote)
                    .foregroundStyle(Tokens.palette.mutedFg)
                if let errorText { Banner(kind: .error, title: LText.verbatim(errorText)) }
                VStack(spacing: Tokens.Spacing.sm) {
                    Button { Task { await submit() } } label: {
                        Label {
                            if model.busy { Text("media.requesting") } else { Text("media.request") }
                        } icon: { Image(systemName: "plus") }
                    }
                    .buttonStyle(.tipsarr(.primary, fullWidth: true))
                    .disabled(model.busy || (isTV && selectedSeasons.isEmpty))
                    Button("common.cancel") { dismiss() }
                        .buttonStyle(.tipsarr(.ghost, fullWidth: true))
                }
            }
            .padding(Tokens.Spacing.xl)
            .padding(.top, Tokens.Spacing.md)
        }
        .scrollBounceBehavior(.basedOnSize)
        .onAppear(perform: setDefaults)
    }

    private var header: some View {
        HStack(spacing: Tokens.Spacing.md) {
            RemoteImage(path: detail?.posterPath, size: .w185) {
                Image(systemName: "film").foregroundStyle(Tokens.palette.mutedFg)
            }
            .frame(width: 56, height: 56 * Tokens.Size.posterRatio)
            .background(Tokens.palette.muted)
            .clipShape(.rect(cornerRadius: Tokens.Radius.sm))
            VStack(alignment: .leading, spacing: 2) {
                Text(verbatim: model.route.title).font(.headline).lineLimit(2)
                Text(verbatim: [detail?.year, isTV ? L10n.string("type.tv") : L10n.string("type.movie")].compactMap { $0 }.joined(separator: " · "))
                    .font(.subheadline)
                    .foregroundStyle(Tokens.palette.mutedFg)
            }
        }
        .accessibilityElement(children: .combine)
    }

    // MARK: Seasons

    private var seasonPicker: some View {
        VStack(alignment: .leading, spacing: Tokens.Spacing.md) {
            HStack {
                Text("req.which_seasons").font(.title3.weight(.semibold))
                Spacer()
                Button {
                    selectedSeasons = allSelected ? [] : Set(selectable.map(\.number))
                } label: {
                    Text(choose(allSelected, "req.select_none", "req.select_all"))
                }
                .font(.subheadline)
                .disabled(selectable.isEmpty)
            }
            ForEach(model.regularSeasons) { season in
                let covered = model.coveredSeasons.contains(season.number)
                Button { toggle(season.number) } label: {
                    HStack(spacing: Tokens.Spacing.md) {
                        Image(systemName: selectedSeasons.contains(season.number) || covered ? "checkmark.circle.fill" : "circle")
                            .font(.title3)
                            .foregroundStyle(covered ? Tokens.palette.mutedFg : Tokens.palette.fg)
                        VStack(alignment: .leading, spacing: 2) {
                            Text(verbatim: season.name).font(.body.weight(.semibold))
                            Text(verbatim: Plural.text("seasons.episodes", count: season.episodeCount))
                                .font(.footnote).foregroundStyle(Tokens.palette.mutedFg)
                        }
                        Spacer()
                        if covered { StatusBadge(state: model.request?.state ?? .requested) }
                    }
                    .padding(Tokens.Spacing.md)
                    .frame(minHeight: Tokens.Size.touchTarget + Tokens.Spacing.md)
                    .background(Tokens.palette.muted, in: .rect(cornerRadius: Tokens.Radius.md))
                    .opacity(covered ? 0.6 : 1)
                    .contentShape(.rect(cornerRadius: Tokens.Radius.md))
                }
                .buttonStyle(.plain)
                .disabled(covered)
                .accessibilityAddTraits(selectedSeasons.contains(season.number) ? .isSelected : [])
            }
        }
    }

    private var allSelected: Bool { !selectable.isEmpty && selectedSeasons.count == selectable.count }

    private func toggle(_ number: Int) {
        if selectedSeasons.contains(number) { selectedSeasons.remove(number) } else { selectedSeasons.insert(number) }
    }

    // MARK: Options

    @ViewBuilder private func optionRows(_ options: RequestOptions) -> some View {
        VStack(spacing: Tokens.Spacing.sm) {
            OptionRow(symbol: "slider.horizontal.3", title: "req.quality_profile") {
                Menu {
                    ForEach(options.profiles) { profile in
                        Button {
                            profileID = profile.id
                        } label: {
                            if profileID == profile.id { Image(systemName: "checkmark") }
                            Text(verbatim: profile.name)
                        }
                    }
                } label: {
                    menuLabel(options.profiles.first { $0.id == profileID }?.name ?? "")
                }
            }
            if model.canChooseFolder, !options.rootFolders.isEmpty {
                OptionRow(symbol: "externaldrive", title: "req.root_folder") {
                    Menu {
                        ForEach(options.rootFolders) { root in
                            Button {
                                folder = root.path
                            } label: {
                                if folder == root.path { Image(systemName: "checkmark") }
                                Text(verbatim: "\(root.path) · \(Self.size(root.freeSpace))")
                            }
                        }
                    } label: {
                        menuLabel(folder ?? options.defaultFolder)
                    }
                }
            }
        }
    }

    /// One line, middle-truncated, with the chevron the system picker would show.
    private func menuLabel(_ text: String) -> some View {
        HStack(spacing: Tokens.Spacing.xs) {
            Text(verbatim: text).lineLimit(1).truncationMode(.middle)
            Image(systemName: "chevron.up.chevron.down").font(.caption2)
        }
        .foregroundStyle(Tokens.palette.mutedFg)
    }

    private static func size(_ bytes: Int64) -> String {
        bytes.formatted(.byteCount(style: .file))
    }

    // MARK: Submit

    private func setDefaults() {
        selectedSeasons = Set(selectable.map(\.number))
        if let options = model.options {
            profileID = options.defaultProfileID
            folder = options.defaultFolder.isEmpty ? options.rootFolders.first?.path : options.defaultFolder
        }
    }

    private func submit() async {
        errorText = nil
        do {
            let seasons = isTV ? selectedSeasons.sorted() : nil
            let record = try await model.submitRequest(
                seasons: seasons,
                profileID: model.options == nil ? nil : profileID,
                folder: model.canChooseFolder ? folder : nil
            )
            onDone(record)
        } catch {
            errorText = APIError.from(error).localizedMessage
        }
    }
}

private struct OptionRow<Control: View>: View {
    let symbol: String
    let title: LText
    @ViewBuilder var control: Control

    var body: some View {
        HStack(spacing: Tokens.Spacing.md) {
            Image(systemName: symbol).foregroundStyle(Tokens.palette.mutedFg).frame(width: 24)
            Text(title).font(.body).lineLimit(1).fixedSize()
            Spacer(minLength: Tokens.Spacing.md)
            control
        }
        .padding(.horizontal, Tokens.Spacing.md)
        .frame(minHeight: Tokens.Size.touchTarget + Tokens.Spacing.md)
        .background(Tokens.palette.muted, in: .rect(cornerRadius: Tokens.Radius.md))
    }
}
