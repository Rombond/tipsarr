import SwiftUI

/// App icons the person can choose from. Add a new one here and as an alternate icon set in the asset catalog.
enum AppIconChoice: CaseIterable, Identifiable {
    case automatic, light, dark, mono

    var id: Self { self }

    /// Name of the alternate icon set; nil is the primary, adaptive icon.
    var iconName: String? {
        switch self {
        case .automatic: nil
        case .light: "AppIconLight"
        case .dark: "AppIconDark"
        case .mono: "AppIconMono"
        }
    }

    var preview: String {
        switch self {
        case .automatic: "IconPreviewDefault"
        case .light: "IconPreviewLight"
        case .dark: "IconPreviewDark"
        case .mono: "IconPreviewMono"
        }
    }

    var title: LText {
        switch self {
        case .automatic: "m.icon.default"
        case .light: "m.icon.light"
        case .dark: "m.icon.dark"
        case .mono: "m.icon.mono"
        }
    }
}

struct AppIconPicker: View {
    @State private var current = AppIconChoice.allCases.first { $0.iconName == UIApplication.shared.alternateIconName } ?? .automatic
    @Environment(ToastCenter.self) private var toast

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: Tokens.Spacing.xl) {
                LazyVGrid(columns: [GridItem(.adaptive(minimum: 120), spacing: Tokens.Spacing.lg)], spacing: Tokens.Spacing.xl) {
                    ForEach(AppIconChoice.allCases) { choice in
                        Button { Task { await select(choice) } } label: {
                            VStack(spacing: Tokens.Spacing.sm) {
                                Image(choice.preview)
                                    .resizable()
                                    .scaledToFit()
                                    .clipShape(.rect(cornerRadius: 26, style: .continuous))
                                    .overlay {
                                        RoundedRectangle(cornerRadius: 26, style: .continuous)
                                            .strokeBorder(current == choice ? Tokens.palette.fg : Tokens.palette.border, lineWidth: current == choice ? 3 : 1)
                                    }
                                    .frame(width: 96, height: 96)
                                HStack(spacing: Tokens.Spacing.xs) {
                                    if current == choice { Image(systemName: "checkmark").font(.footnote.weight(.bold)) }
                                    Text(choice.title).font(.subheadline.weight(current == choice ? .semibold : .regular))
                                }
                            }
                            .frame(maxWidth: .infinity)
                            .contentShape(.rect)
                        }
                        .buttonStyle(.plain)
                        .accessibilityAddTraits(current == choice ? .isSelected : [])
                    }
                }
                Text("m.icon.footer").font(.footnote).foregroundStyle(Tokens.palette.mutedFg)
            }
            .padding(Tokens.Spacing.lg)
        }
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
