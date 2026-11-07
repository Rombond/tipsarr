# Release checklist: mobile apps 1.0

Manual steps for the owner, in order. Prepared files are in `fastlane/metadata/` (listings, review info) and `store/`
(answers). Versions: iOS 1.0 (build 1), Android 1.0.0 (versionCode 10). The server is tagged v1.0.0 separately; the
apps work with any server whose `app.min_version` setting is empty or ≤ 1.0.

## 0. Before anything

- [ ] Demo accounts `admin` / `demo` and `user` / `demo` work from both apps on `https://tipsarr-demo.brebond.com`.
- [ ] Resolve the TODOs in `PRIVACY.md` and merge it to `master` (the privacy URL points there).
- [ ] Take screenshots following `store/screenshots-plan.md`; make the Play feature graphic (1024×500) and 512 icon.

## 1. Google Play (start first: closed testing takes 14 days)

1. [ ] Create the Play Console **personal** developer account (one-time fee, identity verification).
2. [ ] Create the app "Tipsarr", package `com.brebond.tipsarr`, free, app (not game).
3. [ ] **Play App Signing**: enrol, let Google generate the app signing key, and register the existing release key
       (the one in `keystore.properties`) as the **upload key**.
4. [ ] Build the bundle: `cd android && ./gradlew bundleRelease` →
       `app/build/outputs/bundle/release/app-release.aab` (needs `keystore.properties` and `google-services.json`).
5. [ ] Upload to **Internal testing**, install from Play, test push.
6. [ ] **Firebase API key restriction**: copy the *app signing* SHA-1 (Console → Test and release → App integrity)
       into Firebase (Android app fingerprints) and restrict the key in Google Cloud to Android apps
       `com.brebond.tipsarr` + upload SHA-1 + Play signing SHA-1, APIs Firebase Installations + FCM. Retest push.
7. [ ] App content: privacy policy URL, App access (`store/review-notes.md`), Ads: no, Content rating and Target
       audience (`store/age-rating-and-export.md`), Data safety (`store/privacy-answers.md`).
8. [ ] Store listing en-US + fr-FR from `fastlane/metadata/android/<locale>/` (title, short, full description),
       graphics, category *Entertainment*, contact email.
9. [ ] **Closed testing**: at least **12 testers opted in for 14 consecutive days** (personal accounts rule). Keep
       them installed the whole time.
10. [ ] Apply for production access (questionnaire about the test), then promote the build to Production with the
        changelog `changelogs/10.txt`. Any new upload needs versionCode > 10.

## 2. App Store

1. [ ] App Store Connect → new app: bundle id `com.brebond.tipsarr-app`, name "Tipsarr", primary language English
       (U.S.), SKU of your choice, add French (France).
2. [ ] **Production APNs**: archive Release with `/Applications/Xcode.app`; the exported App Store build is signed
       with `aps-environment = production` (the entitlements file says `development`, Xcode rewrites it on export).
       Upload to TestFlight first and confirm a push arrives (relay uses the production APNs endpoint for these
       builds).
3. [ ] Upload build 1.0 (1). Export compliance is answered by `ITSAppUsesNonExemptEncryption = NO`.
4. [ ] App information: category *Entertainment* (secondary *Utilities*), age rating (`store/age-rating-and-export.md`),
       privacy policy URL, App Privacy (`store/privacy-answers.md`, "Data not collected").
5. [ ] Version 1.0 page en-US + fr-FR from `fastlane/metadata/<locale>/` (subtitle, promotional text, description,
       keywords, support and marketing URLs, what's new), screenshots 6.9" + 13" iPad.
6. [ ] App Review information: sign-in `admin` / `demo`, notes from `fastlane/metadata/review_information/notes.txt`,
       your contact details.
7. [ ] Pricing: free, availability: all countries (or choose). Release: manual after approval.
8. [ ] Submit for review.

## 3. After approval

- [ ] Release both apps, tag server v1.0.0, put the store links in `docs/mobile-apps.md` and the README.
- [ ] Later: `fastlane deliver` / `fastlane supply` from CI can reuse `fastlane/metadata` as is (needs an App Store
      Connect API key and a Play service account, not set up).
