# Review notes (App Store and Google Play)

- App Store: `fastlane/metadata/review_information/` (sign-in required: yes, user `admin`, password `demo`). Contact name, email and phone are entered in App Store Connect, not committed.
- Google Play: App content → App access → "All or some functionality is restricted" → add instructions (name "Demo server", username `admin`, password `demo`, and the text below).

> [!NOTE]
> The `admin` / `demo` and `user` / `demo` accounts are being added to the demo gateway: check both work from the apps before submitting.

```text
Tipsarr is a client for a self-hosted server: there is no Tipsarr account or cloud service. To review the app, use our public demo server.

1. Open the app and enter the server address: https://tipsarr-demo.brebond.com
2. Sign in with one of these demo accounts:
   - admin / demo (administrator: can approve requests and see settings)
   - user / demo (regular user)
3. Each sign-in gets a private, throwaway sandbox with sample data. A sandbox is reset after 30 minutes without activity and 60 minutes at most: if the app says the session ended, just sign in again with the same credentials.

The demo never downloads anything (dry-run mode): requests are recorded but nothing is sent to Radarr/Sonarr.

Push notifications are not available on the demo server: on a real server, an administrator must first turn on the push relay in the server settings. Face ID sign-in (iOS) is optional and only keeps the password in the device Keychain.

Source code and privacy policy: https://github.com/Rombond/tipsarr
```
