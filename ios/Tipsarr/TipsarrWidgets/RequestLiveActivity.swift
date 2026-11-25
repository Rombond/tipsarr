import ActivityKit
import SwiftUI
import UIKit
import WidgetKit

/// Lock screen and Dynamic Island for a followed request. Updates come from APNs (push type token).
struct RequestLiveActivity: Widget {
    var body: some WidgetConfiguration {
        ActivityConfiguration(for: RequestActivityAttributes.self) { context in
            LockScreenView(attributes: context.attributes, state: context.state)
                .padding(16)
                .widgetURL(context.attributes.deepLink)
        } dynamicIsland: { context in
            let attributes = context.attributes
            let state = context.state
            return DynamicIsland {
                DynamicIslandExpandedRegion(.leading) {
                    Poster(requestId: attributes.requestId, mediaType: attributes.mediaType)
                        .frame(width: 40, height: 60)
                }
                DynamicIslandExpandedRegion(.trailing) {
                    StageSymbol(stage: state.stage).font(.title2)
                }
                DynamicIslandExpandedRegion(.center) {
                    VStack(alignment: .leading, spacing: 2) {
                        Text(verbatim: attributes.title).font(.headline).lineLimit(1)
                        Text(verbatim: attributes.label(for: state.stage)).font(.subheadline).foregroundStyle(.secondary)
                    }
                    .frame(maxWidth: .infinity, alignment: .leading)
                }
                DynamicIslandExpandedRegion(.bottom) {
                    IslandProgress(state: state)
                }
            } compactLeading: {
                StageSymbol(stage: state.stage)
            } compactTrailing: {
                if state.stage == "downloading" {
                    Text(verbatim: "\(state.percent)%").monospacedDigit()
                } else {
                    Text(verbatim: attributes.label(for: state.stage)).lineLimit(1).frame(maxWidth: 64)
                }
            } minimal: {
                if state.stage == "downloading" {
                    ProgressView(value: Double(state.percent), total: 100).progressViewStyle(.circular)
                } else {
                    StageSymbol(stage: state.stage)
                }
            }
            .widgetURL(attributes.deepLink)
        }
    }
}

private extension RequestActivityAttributes {
    func label(for stage: String) -> String { labels[stage] ?? stage.capitalized }
}

private struct LockScreenView: View {
    let attributes: RequestActivityAttributes
    let state: RequestActivityAttributes.ContentState

    var body: some View {
        HStack(alignment: .center, spacing: 12) {
            Poster(requestId: attributes.requestId, mediaType: attributes.mediaType)
                .frame(width: 48, height: 72)
            VStack(alignment: .leading, spacing: 4) {
                Text(verbatim: attributes.title).font(.headline).lineLimit(2)
                HStack(spacing: 6) {
                    StageSymbol(stage: state.stage)
                    Text(verbatim: attributes.label(for: state.stage)).font(.subheadline)
                }
                ProgressLine(state: state)
            }
            Spacer(minLength: 0)
        }
    }
}

private struct ProgressLine: View {
    let state: RequestActivityAttributes.ContentState

    var body: some View {
        if state.stage == "downloading" {
            VStack(alignment: .leading, spacing: 2) {
                ProgressView(value: Double(state.percent), total: 100)
                HStack {
                    Text(verbatim: "\(state.percent)%").monospacedDigit()
                    Spacer()
                    // Time left as of the last update, not a running clock: between two pushes a countdown
                    // drifts and then counts up once the estimate is past.
                    if state.etaSeconds >= 60 {
                        Text(verbatim: Duration.seconds(state.etaSeconds)
                            .formatted(.units(allowed: [.hours, .minutes], width: .abbreviated)))
                            .monospacedDigit()
                    }
                }
                .font(.caption)
                .foregroundStyle(.secondary)
            }
        }
    }
}

/// The expanded Dynamic Island's bottom row: bar and figures on one line, kept off the rounded corners
/// (a second line under the bar was clipped by them).
private struct IslandProgress: View {
    let state: RequestActivityAttributes.ContentState

    var body: some View {
        if state.stage == "downloading" {
            HStack(spacing: 8) {
                ProgressView(value: Double(state.percent), total: 100)
                Text(verbatim: "\(state.percent)%").monospacedDigit()
                if state.etaSeconds >= 60 {
                    Text(verbatim: Duration.seconds(state.etaSeconds)
                        .formatted(.units(allowed: [.hours, .minutes], width: .abbreviated)))
                        .monospacedDigit()
                        .foregroundStyle(.secondary)
                }
            }
            .font(.caption)
            .padding(.horizontal, 12)
        }
    }
}

private struct StageSymbol: View {
    let stage: String

    var body: some View {
        Image(systemName: symbol).foregroundStyle(color)
    }

    private var symbol: String {
        switch stage {
        case "requested": "clock"
        case "approved": "checkmark.circle"
        case "searching": "magnifyingglass"
        case "downloading": "arrow.down.circle"
        case "available": "checkmark.circle.fill"
        case "declined": "xmark.circle"
        case "failed": "exclamationmark.triangle"
        default: "film"
        }
    }

    private var color: Color {
        switch stage {
        case "available": .green
        case "declined", "failed": .red
        case "downloading": .blue
        default: .orange
        }
    }
}

/// Poster written by the app into the App Group; a symbol when there is none.
private struct Poster: View {
    let requestId: String
    let mediaType: String

    var body: some View {
        Group {
            if let url = LivePoster.url(requestId: requestId), let image = UIImage(contentsOfFile: url.path()) {
                Image(uiImage: image).resizable().scaledToFill()
            } else {
                Image(systemName: mediaType == "tv" ? "tv" : "film")
                    .font(.title3)
                    .foregroundStyle(.secondary)
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
                    .background(.quaternary)
            }
        }
        .clipShape(.rect(cornerRadius: 6))
    }
}
