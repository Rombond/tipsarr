import SwiftUI

/// Building blocks of the Library and Search filter sheets, so the two sheets look and behave the same.

struct FilterSection<Content: View>: View {
    let title: LText
    @ViewBuilder let content: Content

    init(_ title: LText, @ViewBuilder content: () -> Content) {
        self.title = title
        self.content = content()
    }

    var body: some View {
        VStack(alignment: .leading, spacing: Tokens.Spacing.md) {
            Text(title).font(.headline).accessibilityAddTraits(.isHeader)
            content
        }
    }
}

/// Close, reset (icon) and apply buttons of a filter sheet.
private struct FilterSheetToolbar: ViewModifier {
    let canReset: Bool
    let reset: () -> Void
    @Environment(\.dismiss) private var dismiss

    func body(content: Content) -> some View {
        content
            .navigationTitle("library.filters")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItemGroup(placement: .topBarLeading) {
                    Button { dismiss() } label: { Image(systemName: "xmark") }
                        .accessibilityLabel(Text("m.library.close"))
                    Button(action: reset) { Image(systemName: "arrow.counterclockwise") }
                        .disabled(!canReset)
                        .accessibilityLabel(Text("library.reset"))
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button(role: .confirm) { dismiss() } label: { Image(systemName: "checkmark") }
                        .accessibilityLabel(Text("m.library.apply"))
                }
            }
    }
}

extension View {
    func filterSheetToolbar(canReset: Bool, reset: @escaping () -> Void) -> some View {
        modifier(FilterSheetToolbar(canReset: canReset, reset: reset))
    }
}

/// Two rows ("from", "to") in one card; tapping a row unfolds a wheel under it, like the start and end times in Calendar.
struct FilterYearRange: View {
    @Binding var from: Int?
    @Binding var to: Int?
    let range: ClosedRange<Int>
    @State private var open: Field?

    private enum Field { case from, to }

    var body: some View {
        FilterSection("library.year") {
            VStack(spacing: 0) {
                row("library.year_from", field: .from, selection: $from)
                Divider().padding(.leading, Tokens.Spacing.md)
                row("library.year_to", field: .to, selection: $to)
            }
            .background(Tokens.palette.muted, in: .rect(cornerRadius: Tokens.Radius.md))
            .clipShape(.rect(cornerRadius: Tokens.Radius.md))
        }
    }

    private func row(_ title: LText, field: Field, selection: Binding<Int?>) -> some View {
        let isOpen = open == field
        return VStack(spacing: 0) {
            Button {
                withAnimation(.snappy) { open = isOpen ? nil : field }
            } label: {
                HStack {
                    Text(title).foregroundStyle(Tokens.palette.fg)
                    Spacer(minLength: Tokens.Spacing.md)
                    Group {
                        if let year = selection.wrappedValue { Text(verbatim: String(year)) } else { Text("library.any") }
                    }
                    .foregroundStyle(isOpen ? Color.accentColor : Tokens.palette.mutedFg)
                }
                .padding(.horizontal, Tokens.Spacing.md)
                .frame(minHeight: Tokens.Size.touchTarget)
                .contentShape(.rect)
            }
            .buttonStyle(.plain)
            .accessibilityAddTraits(isOpen ? .isSelected : [])

            if isOpen {
                Picker(selection: selection) {
                    Text("library.any").tag(Int?.none)
                    ForEach(Array(range).reversed(), id: \.self) { year in Text(verbatim: String(year)).tag(Int?.some(year)) }
                } label: {
                    Text(title)
                }
                .labelsHidden()
                .pickerStyle(.wheel)
                .frame(height: 150)
                .transition(.opacity.combined(with: .move(edge: .top)))
            }
        }
    }
}

struct FilterMinRating: View {
    @Binding var rating: Double?

    var body: some View {
        FilterSection("library.min_rating") {
            VStack(alignment: .leading, spacing: Tokens.Spacing.xs) {
                Slider(value: Binding(get: { rating ?? 0 }, set: { rating = $0 < 0.5 ? nil : ($0 * 2).rounded() / 2 }), in: 0...9, step: 0.5)
                Group {
                    if let rating { Text(rating, format: .number.precision(.fractionLength(1))) } else { Text("library.any") }
                }
                .font(.footnote)
                .foregroundStyle(Tokens.palette.mutedFg)
            }
        }
    }
}

/// Sort menu; `trailing` holds an extra control next to it (the Library's ascending/descending switch).
struct FilterSort<Option: Hashable, Trailing: View>: View {
    @Binding var selection: Option
    let options: [Option]
    let title: (Option) -> LText
    @ViewBuilder let trailing: Trailing

    var body: some View {
        FilterSection("library.sort_by") {
            HStack(spacing: Tokens.Spacing.md) {
                Picker("library.sort_by", selection: $selection) {
                    ForEach(options, id: \.self) { Text(title($0)).tag($0) }
                }
                .pickerStyle(.menu)
                .frame(maxWidth: .infinity, alignment: .leading)
                trailing
            }
        }
    }
}
