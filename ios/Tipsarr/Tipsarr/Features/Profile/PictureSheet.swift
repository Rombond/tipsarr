import PhotosUI
import SwiftUI

/// Change or remove the profile picture. Max 4 MB: the photo is scaled down before it is sent.
struct PictureSheet: View {
    let profile: Profile
    var onChanged: () -> Void

    @Environment(\.appContext) private var context
    @Environment(\.dismiss) private var dismiss
    @Environment(ToastCenter.self) private var toast
    @State private var choice: PhotosPickerItem?
    @State private var working = false

    var body: some View {
        VStack(spacing: Tokens.Spacing.lg) {
            Text("m.profile.picture_title").font(.title3.weight(.semibold))
            VStack(spacing: Tokens.Spacing.sm) {
                PhotosPicker(selection: $choice, matching: .images) {
                    Label { Text("m.picture.library") } icon: { Image(systemName: "photo") }
                        .frame(maxWidth: .infinity)
                }
                .buttonStyle(.tipsarr(.secondary, fullWidth: true))
                Button { Task { await remove() } } label: {
                    Label { Text("profile.remove_picture") } icon: { Image(systemName: "trash") }
                        .frame(maxWidth: .infinity)
                }
                .buttonStyle(.tipsarr(.secondary, fullWidth: true))
                Button("common.cancel") { dismiss() }.buttonStyle(.tipsarr(.tonal, fullWidth: true))
            }
            .disabled(working)
            Text("m.picture.footer").font(.footnote).foregroundStyle(Tokens.palette.mutedFg).multilineTextAlignment(.center)
        }
        .padding(Tokens.Spacing.xl)
        .onChange(of: choice) { _, item in
            guard let item else { return }
            Task { await upload(item) }
        }
    }

    private func upload(_ item: PhotosPickerItem) async {
        guard let context else { return }
        working = true
        defer { working = false }
        do {
            guard let data = try await item.loadTransferable(type: Data.self), let jpeg = Self.scaledJPEG(data) else {
                toast.show(L10n.string("profile.picture_invalid"), kind: .error)
                return
            }
            try await context.api.uploadAvatar(jpeg: jpeg)
            toast.show(L10n.string("profile.picture_saved"))
            onChanged()
            dismiss()
        } catch {
            toast.show(APIError.from(error).localizedMessage, kind: .error)
        }
    }

    private func remove() async {
        guard let context else { return }
        working = true
        defer { working = false }
        do {
            try await context.api.deleteAvatar()
            onChanged()
            dismiss()
        } catch {
            toast.show(APIError.from(error).localizedMessage, kind: .error)
        }
    }

    /// 768 px on the long side, JPEG: well under the 4 MB limit.
    private static func scaledJPEG(_ data: Data) -> Data? {
        guard let image = UIImage(data: data) else { return nil }
        let longSide = max(image.size.width, image.size.height)
        let scale = min(1, 768 / longSide)
        let size = CGSize(width: image.size.width * scale, height: image.size.height * scale)
        let format = UIGraphicsImageRendererFormat.default()
        format.scale = 1
        let resized = UIGraphicsImageRenderer(size: size, format: format).image { _ in image.draw(in: CGRect(origin: .zero, size: size)) }
        return resized.jpegData(compressionQuality: 0.85)
    }
}
