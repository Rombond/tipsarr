import SwiftUI

/// Compact genre control: one row with the selection summary, which opens a searchable multi-select list.
/// Used by the Library and Search filter sheets (it needs the sheet's navigation stack).
struct GenrePickerRow: View {
    let names: [String]
    @Binding var selection: Set<String>
    @Binding var anyGenre: Bool
    /// More than this many genres at once make a search too narrow to be useful.
    var limit = 6

    var body: some View {
        NavigationLink {
            GenrePickerList(names: names, selection: $selection, anyGenre: $anyGenre, limit: limit)
        } label: {
            HStack(spacing: Tokens.Spacing.md) {
                Text(verbatim: summary)
                    .font(.body)
                    .foregroundStyle(selection.isEmpty ? Tokens.palette.mutedFg : Tokens.palette.fg)
                    .lineLimit(1)
                Spacer(minLength: 0)
                Image(systemName: "chevron.right").font(.footnote.weight(.semibold)).foregroundStyle(Tokens.palette.mutedFg)
            }
            .padding(.horizontal, Tokens.Spacing.md)
            .frame(minHeight: Tokens.Size.touchTarget)
            .background(Tokens.palette.muted, in: .rect(cornerRadius: Tokens.Radius.md))
            .contentShape(.rect(cornerRadius: Tokens.Radius.md))
        }
        .buttonStyle(.plain)
        .accessibilityLabel(Text("library.genres"))
        .accessibilityValue(Text(verbatim: summary))
    }

    private var summary: String {
        let sorted = selection.sorted { $0.localizedStandardCompare($1) == .orderedAscending }
        switch sorted.count {
        case 0: return LText("m.library.genres_any").resolved
        case 1: return sorted[0]
        default: return L10n.string("m.library.genres_more", sorted[0], sorted.count - 1)
        }
    }
}

struct GenrePickerList: View {
    let names: [String]
    @Binding var selection: Set<String>
    @Binding var anyGenre: Bool
    let limit: Int
    @State private var query = ""

    private var shown: [String] {
        let text = query.trimmingCharacters(in: .whitespacesAndNewlines)
        return text.isEmpty ? names : names.filter { $0.localizedStandardContains(text) }
    }

    var body: some View {
        List {
            if selection.count > 1 {
                Section {
                    Picker("library.genres", selection: $anyGenre) {
                        Text("m.library.genre_all").tag(false)
                        Text("m.library.genre_any").tag(true)
                    }
                    .pickerStyle(.segmented)
                }
            }
            Section {
                ForEach(shown, id: \.self) { name in
                    let selected = selection.contains(name)
                    Button {
                        if selected { selection.remove(name) } else if selection.count < limit { selection.insert(name) }
                    } label: {
                        HStack {
                            Text(verbatim: name).foregroundStyle(Tokens.palette.fg)
                            Spacer(minLength: 0)
                            if selected { Image(systemName: "checkmark").foregroundStyle(Tokens.palette.primary) }
                        }
                        .frame(minHeight: Tokens.Size.touchTarget)
                        .contentShape(.rect)
                    }
                    .buttonStyle(.plain)
                    .accessibilityAddTraits(selected ? .isSelected : [])
                }
            }
        }
        .navigationTitle("m.library.genres_pick")
        .navigationBarTitleDisplayMode(.inline)
        .searchable(text: $query, prompt: Text("m.library.genres_search"))
        .toolbar {
            if !selection.isEmpty {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("library.reset") { selection = [] }
                }
            }
        }
    }
}
