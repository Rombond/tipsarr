import SwiftUI

struct ProgressBar: View {
    /// 0...100
    let percent: Int
    var tint: Color = Tokens.Status.downloading

    var body: some View {
        GeometryReader { proxy in
            ZStack(alignment: .leading) {
                Capsule().fill(Tokens.palette.muted)
                Capsule().fill(tint).frame(width: proxy.size.width * CGFloat(min(max(percent, 0), 100)) / 100)
            }
        }
        .frame(height: 6)
        .accessibilityElement()
        .accessibilityValue(Text("\(percent)%"))
    }
}
