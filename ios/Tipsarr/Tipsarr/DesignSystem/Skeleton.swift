import SwiftUI

/// Placeholder shimmer. Static when Reduce Motion is on.
private struct SkeletonModifier: ViewModifier {
    @Environment(\.accessibilityReduceMotion) private var reduceMotion
    @State private var phase = false

    func body(content: Content) -> some View {
        content
            .redacted(reason: .placeholder)
            .opacity(reduceMotion ? 0.6 : (phase ? 0.4 : 0.8))
            .animation(reduceMotion ? nil : .easeInOut(duration: 0.9).repeatForever(autoreverses: true), value: phase)
            .onAppear { phase = true }
            .accessibilityElement(children: .ignore)
            .accessibilityLabel(Text("common.loading"))
    }
}

extension View {
    func skeleton() -> some View { modifier(SkeletonModifier()) }
}

struct SkeletonBlock: View {
    var height: CGFloat = 16
    var cornerRadius: CGFloat = Tokens.Radius.sm

    var body: some View {
        RoundedRectangle(cornerRadius: cornerRadius)
            .fill(Tokens.palette.border)
            .frame(height: height)
    }
}

struct PosterSkeleton: View {
    var body: some View {
        VStack(alignment: .leading, spacing: Tokens.Spacing.sm) {
            Color.clear
                .aspectRatio(1 / Tokens.Size.posterRatio, contentMode: .fit)
                .background(Tokens.palette.border, in: .rect(cornerRadius: Tokens.Radius.md))
            SkeletonBlock(height: 14)
            SkeletonBlock(height: 12).frame(maxWidth: 60)
        }
        .skeleton()
    }
}

#Preview {
    LazyVGrid(columns: Array(repeating: GridItem(.flexible(), spacing: Tokens.Spacing.md), count: 3), spacing: Tokens.Spacing.lg) {
        ForEach(0..<6, id: \.self) { _ in PosterSkeleton() }
    }
    .padding()
}
