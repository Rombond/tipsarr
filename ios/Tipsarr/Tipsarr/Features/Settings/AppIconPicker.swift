import SwiftUI

/// App icons the person can choose from. Add a new one here and as an alternate icon set in the asset catalog.
enum AppIconChoice: CaseIterable, Identifiable {
    case light, dark, mono

    var id: Self { self }

    /// Name of the alternate icon set.
    var iconName: String? {
        switch self {
        case .light: "AppIconLight"
        case .dark: "AppIconDark"
        case .mono: "AppIconMono"
        }
    }

    var preview: String {
        switch self {
        case .light: "IconPreviewLight"
        case .dark: "IconPreviewDark"
        case .mono: "IconPreviewMono"
        }
    }

    var title: LText {
        switch self {
        case .light: "m.icon.light"
        case .dark: "m.icon.dark"
        case .mono: "m.icon.mono"
        }
    }
}

struct AppIconPicker: View {
    @State private var current = AppIconChoice.allCases.first { $0.iconName == UIApplication.shared.alternateIconName } ?? .light
    @Environment(ToastCenter.self) private var toast

    var body: some View {
        List {
            Section {
                ForEach(AppIconChoice.allCases) { choice in
                    Button { Task { await select(choice) } } label: {
                        HStack(spacing: Tokens.Spacing.md) {
                            Image(choice.preview)
                                .resizable()
                                .scaledToFit()
                                .frame(width: 48, height: 48)
                                .clipShape(.rect(cornerRadius: 11, style: .continuous))
                                .overlay { RoundedRectangle(cornerRadius: 11, style: .continuous).strokeBorder(Tokens.palette.border) }
                            Text(choice.title).foregroundStyle(Tokens.palette.fg)
                            Spacer()
                            if current == choice { Image(systemName: "checkmark").foregroundStyle(Tokens.palette.fg) }
                        }
                        .contentShape(.rect)
                    }
                    .accessibilityAddTraits(current == choice ? .isSelected : [])
                }
            } footer: {
                Text("m.icon.footer")
            }
        }
        .scrollContentBackground(.hidden)
        .readableColumn()
        .background(Tokens.palette.bg)
        .navigationTitle("m.settings.app_icon")
        .navigationBarTitleDisplayMode(.inline)
    }

    private func select(_ choice: AppIconChoice) async {
        guard choice != current else { return }
        do {
            try await UIApplication.shared.setAlternateIconName(choice.iconName)
            current = choice
        } catch {
            toast.show(error.localizedDescription, kind: .error)
        }
    }
}
