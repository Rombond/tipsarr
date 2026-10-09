import LocalAuthentication

enum DeviceAuth {
    /// The iPhone has a passcode: needed to store the login behind Face ID.
    static var isAvailable: Bool {
        LAContext().canEvaluatePolicy(.deviceOwnerAuthentication, error: nil)
    }
}
