import SwiftUI

/// Full-screen lock shown over everything while the app is locked.
struct LockView: View {
    let lock: AppLock
    @Environment(\.scenePhase) private var scenePhase

    var body: some View {
        VStack(spacing: Tokens.Spacing._2xl) {
            Spacer()
            Image(systemName: lock.symbol)
                .font(.system(size: 56, weight: .light))
                .foregroundStyle(Tokens.palette.fg)
                .accessibilityHidden(true)
            VStack(spacing: Tokens.Spacing.sm) {
                Text("m.lock.title").font(.title2.weight(.bold)).accessibilityAddTraits(.isHeader)
                Text("m.lock.subtitle").font(.subheadline).foregroundStyle(Tokens.palette.mutedFg)
            }
            .multilineTextAlignment(.center)
            Button { Task { await lock.unlock() } } label: {
                Label { Text("m.lock.unlock") } icon: { Image(systemName: lock.symbol) }
            }
            .buttonStyle(.tipsarr(.primary))
            Spacer()
        }
        .padding(Tokens.Spacing._2xl)
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .background(Tokens.palette.bg.ignoresSafeArea())
        // Ask as soon as the lock is on screen and the app is active.
        .task(id: scenePhase) { if scenePhase == .active { await lock.unlock() } }
    }
}

/// Hides the content in the app switcher while the lock is on.
struct PrivacyCover: View {
    var body: some View {
        ZStack {
            Tokens.palette.bg.ignoresSafeArea()
            Image(systemName: "lock.fill").font(.largeTitle).foregroundStyle(Tokens.palette.mutedFg)
        }
    }
}
