import SwiftUI

struct PersonScreen: View {
    let route: PersonRoute
    @Environment(\.appContext) private var context
    @State private var person: PersonDetail?
    @State private var error: APIError?
    @State private var expanded = false

    var body: some View {
        Group {
            if let person {
                content(person)
            } else if let error {
                ErrorState(error: error) { await load() }
            } else {
                ProgressView().frame(maxWidth: .infinity, maxHeight: .infinity)
            }
        }
        .background(Tokens.palette.bg)
        .navigationTitle(Text(verbatim: route.name))
        .navigationBarTitleDisplayMode(.inline)
        .task { await load() }
    }

    private func load() async {
        guard let context else { return }
        do {
            person = try await context.api.person(id: route.id)
            error = nil
        } catch {
            self.error = APIError.from(error)
        }
    }

    private func content(_ person: PersonDetail) -> some View {
        ScrollView {
            VStack(alignment: .leading, spacing: Tokens.Spacing.xl) {
                HStack(spacing: Tokens.Spacing.lg) {
                    PersonPhoto(path: person.profilePath).frame(width: 96, height: 96)
                    VStack(alignment: .leading, spacing: Tokens.Spacing.xs) {
                        Text(verbatim: person.name).font(.title2.weight(.bold)).accessibilityAddTraits(.isHeader)
                        if let department = person.department, !department.isEmpty {
                            Text(verbatim: department).font(.subheadline).foregroundStyle(Tokens.palette.mutedFg)
                        }
                        if let born = person.birthday.flatMap(Self.date) {
                            Text(verbatim: L10n.string("m.person.born", born)).font(.footnote).foregroundStyle(Tokens.palette.mutedFg)
                        }
                        if let died = person.deathday.flatMap(Self.date) {
                            Text(verbatim: L10n.string("m.person.died", died)).font(.footnote).foregroundStyle(Tokens.palette.mutedFg)
                        }
                    }
                }
                if let biography = person.biography {
                    VStack(alignment: .leading, spacing: Tokens.Spacing.sm) {
                        Text("m.person.biography").font(.title3.weight(.bold)).accessibilityAddTraits(.isHeader)
                        Text(verbatim: biography).font(.callout).lineSpacing(3).lineLimit(expanded ? nil : 6)
                        Button { expanded.toggle() } label: { Text(choose(expanded, "m.common.less", "m.common.more")) }.font(.subheadline)
                    }
                }
                if !person.credits.isEmpty {
                    VStack(alignment: .leading, spacing: Tokens.Spacing.md) {
                        Text("m.person.known_for").font(.title3.weight(.bold)).accessibilityAddTraits(.isHeader)
                        LazyVGrid(columns: [GridItem(.adaptive(minimum: 104, maximum: 180), spacing: Tokens.Spacing.md, alignment: .top)], spacing: Tokens.Spacing.lg) {
                            ForEach(person.credits) { item in
                                NavigationLink(value: item.route) {
                                    PosterCard(title: item.title, subtitle: item.releaseYear, posterPath: item.posterPath, state: item.state)
                                }
                                .buttonStyle(.plain)
                            }
                        }
                    }
                }
            }
            .padding(Tokens.Spacing.lg)
            .frame(maxWidth: 720)
            .frame(maxWidth: .infinity)
        }
    }

    private static func date(_ text: String) -> String? {
        (try? Date(text, strategy: .iso8601.year().month().day()))?.formatted(date: .long, time: .omitted)
    }
}
