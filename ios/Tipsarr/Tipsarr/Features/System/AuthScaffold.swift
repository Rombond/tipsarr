import SwiftUI

/// Shared layout of the connect / login / system screens: centred column, readable on iPad.
struct AuthScaffold<Content: View>: View {
    let symbol: String
    let title: LocalizedStringResource
    var subtitle: LocalizedStringResource?
    @ViewBuilder var content: Content

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: Tokens.Spacing._2xl) {
                VStack(alignment: .leading, spacing: Tokens.Spacing.md) {
                    Image(systemName: symbol)
                        .font(.title)
                        .foregroundStyle(Tokens.palette.primaryFg)
                        .frame(width: 56, height: 56)
                        .background(Tokens.palette.primary, in: .rect(cornerRadius: Tokens.Radius.lg))
                        .accessibilityHidden(true)
                    Text(title)
                        .font(.largeTitle.weight(.bold))
                        .accessibilityAddTraits(.isHeader)
                    if let subtitle {
                        Text(subtitle).font(.body).foregroundStyle(Tokens.palette.mutedFg)
                    }
                }
                content
            }
            .frame(maxWidth: 480, alignment: .leading)
            .padding(Tokens.Spacing._2xl)
            .frame(maxWidth: .infinity)
        }
        .scrollBounceBehavior(.basedOnSize)
        .background(Tokens.palette.bg)
    }
}
