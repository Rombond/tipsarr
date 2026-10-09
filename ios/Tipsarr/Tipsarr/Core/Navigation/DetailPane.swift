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
    @ViewBuilder var label: Label
    @Environment(\.detailPane) private var pane

    var body: some View {
        if let pane {
            Button { pane.content = .media(route) } label: { label }
        } else {
            NavigationLink(value: route) { label }
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
