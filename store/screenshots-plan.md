# Screenshot plan

Take every screenshot against the demo server `https://tipsarr-demo.brebond.com` (sample data, no real account,
library or server address). Use the `admin` account for the request-approval shot, `user` for the rest. English for
`en-US`, French (device language) for `fr-FR`. Light theme, except one dark shot.

Existing: **none** in the repo (no `store-screenshots/` folder). Everything below is missing.

## Sizes

| Store | Device class | Size (portrait) | Required | Count |
|---|---|---|---|---|
| App Store | iPhone 6.9" (16/17 Pro Max) | 1320 × 2868 | yes (covers smaller iPhones) | 3 to 10 |
| App Store | iPad 13" (iPad Pro M4/M5) | 2064 × 2752 | yes (app supports iPad) | 3 to 10 |
| Google Play | Phone | 1080 × 2400 (9:20, each side 320 to 3840) | yes, min 2 | 4 to 8 |
| Google Play | 7" tablet | e.g. 1200 × 1920 | recommended (needed for tablet featuring) | 4 to 8 |
| Google Play | 10" tablet | e.g. 1600 × 2560 | recommended | 4 to 8 |
| Google Play | Feature graphic | 1024 × 500 | **yes** | 1 |
| Google Play | App icon | 512 × 512 PNG | **yes** | 1 (from `design/icon/`) |

fastlane paths: `fastlane/screenshots/<locale>/` (iOS, deliver detects the device by size) and
`fastlane/metadata/android/<locale>/images/{phoneScreenshots,sevenInchScreenshots,tenInchScreenshots}/`,
`images/featureGraphic.png`, `images/icon.png`.

## Screens (same order on every device)

1. **Discover**: trending row and suggestions, availability badges visible.
2. **Title page**: a movie with backdrop, rating, cast and the Request button.
3. **Request sheet**: show request with season selection, quality profile and folder.
4. **Requests**: list with progress (requested, downloading, available).
5. **Search**: keyword search results ("shark") or genre landing.
6. **Profile / statistics**: watch-time and genre charts.
7. **Admin approval** (admin account): pending requests with approve/decline.
8. **Teasers** (optional): trailer feed, only if enabled on the demo.
9. **Dark theme** version of Discover.
10. **Sign-in screen** with the server address field (shows "bring your own server").

iPad / tablets: use the two-column layouts (Settings, title page centered, Library with larger posters).

> [!WARNING]
> Check every frame for the session timer, real names, a private hostname or a notification banner before uploading.
> Mature posters in the trending row can affect the age rating review: pick a row without explicit artwork.
