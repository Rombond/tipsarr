import SwiftUI

@main
struct TipsarrApp: App {
    var body: some Scene {
        WindowGroup {
            // Step 1 root: the component catalogue. Replaced by RootView (launch flow) in step 2.
            Catalogue()
        }
    }
}
