import SwiftUI

/// Every foundation component on one screen. Preview only; not shipped in a screen.
struct Catalogue: View {
    @State private var address = ""
    @State private var showSheet = false

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: Tokens.Spacing._2xl) {
                section("Ratings") {
                    RatingsStrip(ratings: .init(tmdb: 8.2, imdb: 8.5, rottenTomatoes: 92, metacritic: 79))
                    RatingsStrip(ratings: .init(tmdb: 8.7, rottenTomatoes: 41, metacritic: 35))
                    HStack(spacing: Tokens.Spacing.lg) {
                        ProviderMark(source: .tmdb, value: 8.2)
                        ProviderMark(source: .imdb, value: 8.5)
                        ProviderMark(source: .rottenTomatoes, value: 92)
                        ProviderMark(source: .rottenTomatoes, value: 41)
                        MetacriticSquare(value: 79)
                        MetacriticSquare(value: 45)
                        MetacriticSquare(value: 20)
                    }
                }
                section("Posters") {
                    LazyVGrid(columns: Array(repeating: GridItem(.flexible(), spacing: Tokens.Spacing.md), count: 3), spacing: Tokens.Spacing.lg) {
                        PosterCard(title: "Blade Runner 2049", subtitle: "2017", state: .available)
                        PosterCard(title: "Severance", subtitle: "2022", state: .downloading)
                        PosterCard(title: "A very long title that gets truncated", subtitle: "1999", state: .partial)
                        PosterSkeleton()
                    }
                }
                section("Badges") {
                    FlowRow { ForEach(RequestState.allCases, id: \.self) { StatusBadge(state: $0) } }
                }
                section("Buttons") {
                    Button("media.request") {}.buttonStyle(.tipsarr(.primary, fullWidth: true))
                    Button("common.cancel") {}.buttonStyle(.tipsarr(.secondary, fullWidth: true))
                    Button("common.delete") {}.buttonStyle(.tipsarr(.destructive, fullWidth: true))
                    Button("common.save") {}.buttonStyle(.tipsarr(.primary, fullWidth: true)).disabled(true)
                }
                section("Chips") {
                    HStack {
                        Chip(title: "m.tab.discover", isSelected: true)
                        Chip(title: "m.tab.library")
                    }
                }
                section("Field") {
                    Field(title: "m.connect.address", text: $address, prompt: "m.connect.placeholder")
                    Field(title: "m.connect.address", text: $address, error: "m.connect.unreachable")
                }
                section("Banner") {
                    Banner(title: "banner.dry_run", message: "banner.dry_run_long")
                    Banner(kind: .error, title: "m.offline.title", message: "m.offline.body", onDismiss: {})
                }
                section("Sheet") {
                    Button("common.details") { showSheet = true }.buttonStyle(.tipsarr(.secondary))
                }
            }
            .padding(Tokens.Spacing.lg)
        }
        .background(Tokens.palette.bg)
        .sheet(isPresented: $showSheet) {
            StateView.noResults().tipsarrSheet()
        }
    }

    private func section<Content: View>(_ name: String, @ViewBuilder content: () -> Content) -> some View {
        VStack(alignment: .leading, spacing: Tokens.Spacing.md) {
            Text(verbatim: name).font(.footnote.weight(.semibold)).foregroundStyle(Tokens.palette.mutedFg)
            content()
        }
    }
}

private struct FlowRow<Content: View>: View {
    @ViewBuilder var content: Content
    var body: some View {
        ViewThatFits(in: .horizontal) {
            HStack { content }
            VStack(alignment: .leading) { content }
        }
    }
}

#Preview("Light") { Catalogue() }
#Preview("Dark") { Catalogue().preferredColorScheme(.dark) }
#Preview("fr XXL") {
    Catalogue().environment(\.locale, .init(identifier: "fr")).dynamicTypeSize(.xxxLarge)
}
