import SwiftUI

/// Browsers and apps where the account is signed in.
struct DevicesView: View {
    @Environment(\.appContext) private var context
    @Environment(ToastCenter.self) private var toast
    @State private var devices: [DeviceSession] = []
    @State private var phase: Phase = .loading
    @State private var confirmOthers = false

    enum Phase: Equatable { case loading, loaded, failed(APIError) }

    var body: some View {
        List {
            Section {
                switch phase {
                case .loading:
                    ProgressView().frame(maxWidth: .infinity)
                case .failed(let error):
                    ErrorState(error: error) { await load() }
                case .loaded:
                    ForEach(devices) { device in
                        row(device)
                            .swipeActions(edge: .trailing, allowsFullSwipe: false) {
                                if !device.current {
                                    Button(role: .destructive) { Task { await revoke(device) } } label: {
                                        Label("profile.device_sign_out", systemImage: "rectangle.portrait.and.arrow.right")
                                    }
                                }
                            }
                    }
                }
            } header: {
                Text("profile.devices_desc").textCase(nil)
            }
            if devices.contains(where: { !$0.current }) {
                Section {
                    Button(role: .destructive) { confirmOthers = true } label: {
                        Text("profile.devices_sign_out_others").frame(maxWidth: .infinity, alignment: .center)
                    }
                        .confirmationDialog(Text("profile.devices_sign_out_others"), isPresented: $confirmOthers, titleVisibility: .visible) {
                            Button("profile.devices_sign_out_others", role: .destructive) { Task { await revokeOthers() } }
                            Button("common.cancel", role: .cancel) {}
                        }
                }
            }
        }
        .scrollContentBackground(.hidden)
        .background(Tokens.palette.bg)
        .navigationTitle("profile.devices_title")
        .navigationBarTitleDisplayMode(.inline)
        .task { await load() }
        .refreshable { await load() }
    }

    private func row(_ device: DeviceSession) -> some View {
        HStack(spacing: Tokens.Spacing.md) {
            Image(systemName: device.platform == .web ? "globe" : "iphone")
                .frame(width: 28)
                .foregroundStyle(Tokens.palette.mutedFg)
            VStack(alignment: .leading, spacing: 2) {
                HStack(spacing: Tokens.Spacing.sm) {
                    Text(verbatim: device.deviceName.isEmpty ? L10n.string("profile.device_web") : device.deviceName).font(.body.weight(.medium))
                    if device.current {
                        Text("profile.device_current")
                            .font(.caption2.weight(.semibold))
                            .padding(.horizontal, Tokens.Spacing.sm)
                            .padding(.vertical, 2)
                            .background(Tokens.Status.available.opacity(0.15), in: .capsule)
                            .foregroundStyle(Tokens.Status.available)
                    }
                }
                Text(verbatim: L10n.string("profile.device_last_used", device.lastSeen.formatted(.relative(presentation: .named))))
                    .font(.footnote).foregroundStyle(Tokens.palette.mutedFg)
            }
        }
        .accessibilityElement(children: .combine)
    }

    private func load() async {
        guard let context else { return }
        do {
            devices = try await context.api.sessions().sorted { $0.lastSeen > $1.lastSeen }
            phase = .loaded
        } catch {
            if devices.isEmpty { phase = .failed(APIError.from(error)) }
        }
    }

    private func revoke(_ device: DeviceSession) async {
        guard let context else { return }
        do {
            try await context.api.revokeSession(id: device.id)
            devices.removeAll { $0.id == device.id }
            toast.show(L10n.string("profile.device_signed_out"))
        } catch {
            toast.show(APIError.from(error).localizedMessage, kind: .error)
        }
    }

    private func revokeOthers() async {
        guard let context else { return }
        do {
            try await context.api.revokeOtherSessions()
            devices.removeAll { !$0.current }
            toast.show(L10n.string("profile.devices_signed_out_others"))
        } catch {
            toast.show(APIError.from(error).localizedMessage, kind: .error)
        }
    }
}
