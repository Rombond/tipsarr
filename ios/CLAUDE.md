# iOS app

SwiftUI, project `Tipsarr/Tipsarr.xcodeproj`, sources in `Tipsarr/Tipsarr/` (`App/`, `Core/`, `DesignSystem/`, `Features/`). Building needs a Mac with Xcode: on Linux, edit only and say it is unbuilt.

- API client: swift-openapi-generator build plugin reads `Tipsarr/Core/API/openapi.yaml` (+ `openapi-generator-config.yaml`). That yaml is a copy made by `make generate` from `api/openapi.yaml` with `additionalProperties: false` stripped, so an older app decodes a newer server. Never edit it by hand; hand-written wrappers are `TipsarrAPI+*.swift` and `*Models.swift`.
- `DesignSystem/Tokens.swift` is copied by `make generate`. `Localizable.xcstrings` and `InfoPlist.xcstrings` come from `make ios-strings` (from `locales/`), never edited in place.
- The project file `project.xcproj` is in JSON format. Version: `MARKETING_VERSION` (user version) and `CURRENT_PROJECT_VERSION` (build number, bump for every upload) there.
- Push: `Tipsarr.entitlements` holds `aps-environment`; registration goes through `TipsarrAPI+Push.swift` to the server, which talks to the push relay.
- Layouts: iPhone tabs (`App/MainTabView.swift`), iPad and dual-screen/foldable two-pane shell (`App/Shell/DualPaneShell.swift`, `Core/Layout/AdaptiveLayout.swift`, `FoldInfo.swift`). Check both when changing a screen.
