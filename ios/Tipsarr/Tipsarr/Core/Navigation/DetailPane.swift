import SwiftUI

/// The right pane of the unfolded iPhone Duo. When it exists (injected by the dual-pane shell), a tap on a
/// title or a request shows it there, beside the list, instead of pushing it onto the left pane.
@MainActor @Observable
final class DetailPane {
    enum Content: Hashable {
        case media(MediaRoute)
        case request(RequestRecord)
        case profile(ProfileRoute)
    }

    var content: Content?
    /// Lets the detail pane build a request detail (set by the Requests screen).
    var requestsModel: RequestsModel?
}

private struct DetailPaneKey: EnvironmentKey {
    static let defaultValue: DetailPane? = nil
}

extension EnvironmentValues {
    var detailPane: DetailPane? {
        get { self[DetailPaneKey.self] }
        set { self[DetailPaneKey.self] = newValue }
    }
}

/// Opens a title: pushed on the current stack, or shown in the right pane when there is one.
struct MediaLink<Label: View>: View {
    let route: MediaRoute
    /// Show "Request" in the long-press menu (the title has no request yet).
    var requestable = false
    @ViewBuilder var label: Label
    @Environment(\.detailPane) private var pane
    @Environment(\.appContext) private var context
    @Environment(ToastCenter.self) private var toast

    var body: some View {
        Group {
            if let pane {
                Button { pane.content = .media(route) } label: { label }
            } else {
                NavigationLink(value: route) { label }
            }
        }
        .contextMenu {
            if requestable {
                Button { run("m.menu.requested") { _ = try await $0.createRequest(route.type, tmdbId: route.tmdbId, seasons: nil, profileID: nil, folder: nil) } } label: {
                    SwiftUI.Label { Text("media.request") } icon: { Image(systemName: "plus") }
                }
            }
            Button { run("m.menu.watchlisted") { try await $0.setWatchlisted(true, route.type, id: route.tmdbId) } } label: {
                SwiftUI.Label { Text("actions.watchlist") } icon: { Image(systemName: "bookmark") }
            }
            Button { run("m.menu.hidden") { try await $0.setBlocklisted(true, route.type, id: route.tmdbId) } } label: {
                SwiftUI.Label { Text("media.not_interested") } icon: { Image(systemName: "eye.slash") }
            }
        }
    }

    /// Long-press actions: done with the title's defaults, the result shown as a toast.
    private func run(_ done: LText, _ action: @escaping (TipsarrAPI) async throws -> Void) {
        guard let api = context?.api else { return }
        Task {
            do {
                try await action(api)
                toast.show(done.resolved)
            } catch {
                toast.show(APIError.from(error).localizedMessage, kind: .error)
            }
        }
    }
}

/// Opens a request the same way.
struct RequestLink<Label: View>: View {
    let record: RequestRecord
    @ViewBuilder var label: Label
    @Environment(\.detailPane) private var pane

    var body: some View {
        if let pane {
            Button { pane.content = .request(record) } label: { label }
        } else {
            NavigationLink(value: record) { label }
        }
    }
}

/// Opens a Profile or Settings screen: pushed on the current stack, or shown in the right pane on the Duo.
/// `chevron` adds the disclosure arrow that a list row gets from a NavigationLink.
struct ProfileLink<Label: View>: View {
    let route: ProfileRoute
    var chevron = false
    @ViewBuilder var label: Label
    @Environment(\.detailPane) private var pane

    var body: some View {
        if let pane {
            Button { pane.content = .profile(route) } label: {
                HStack {
                    label
                    if chevron { Spacer(minLength: 0); Image(systemName: "chevron.right").font(.footnote.weight(.semibold)).foregroundStyle(Tokens.palette.mutedFg) }
                }
                .contentShape(.rect)
            }
            .buttonStyle(.plain)
        } else {
            NavigationLink(value: route) { label }
        }
    }
}

/// Pickers push a list on iPhone; next to a detail pane they open a menu in place instead.
struct AdaptivePickerStyle: ViewModifier {
    @Environment(\.detailPane) private var pane

    func body(content: Content) -> some View {
        if pane != nil { content.pickerStyle(.menu) } else { content.pickerStyle(.navigationLink) }
    }
}
