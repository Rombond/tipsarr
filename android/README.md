# Tipsarr Android

Kotlin + Jetpack Compose + Material 3 (minSdk 26, compileSdk 37). It follows the iOS app: same behaviour, same screens, similar UI (`../ios/`). Same API (`../api/openapi.yaml`, copied by `make generate` to `app/src/main/openapi/`), same tokens (`../design/tokens.json`, generated to `app/.../design/Tokens.kt`), same strings (`app/src/main/res/values*/strings.xml`, generated from `../locales`). Icon layers come from `../design/icon/Android-*`.

Build and run (Android Studio's bundled JDK, SDK in `~/Library/Android/sdk`):

```sh
export JAVA_HOME="/Applications/Android Studio.app/Contents/jbr/Contents/Home"
export ANDROID_HOME=$HOME/Library/Android/sdk
echo "sdk.dir=$ANDROID_HOME" > local.properties     # once, not committed
./gradlew installDebug                               # with an emulator or phone connected
```

`make generate` refreshes the generated files. Do not edit `Tokens.kt`, `strings.xml` or `openapi.yaml` by hand.

## Push (Firebase)

`app/google-services.json` (from your own Firebase project, package `com.brebond.tipsarr`) is not committed and the build needs it. Push goes through the relay (`push-relay` repo) with a Firebase service account; the app only needs the token.

## Release build

```sh
./gradlew assembleRelease     # app/build/outputs/apk/release/app-release.apk, signed with the release key
```

Signing reads `keystore.properties` (not committed) which points at the keystore (`~/.tipsarr-signing/tipsarr-release.jks`). **Back up the keystore and its passwords: an app signed with another key cannot update an installed one.** Without `keystore.properties` the release APK is unsigned. Install on a phone with `adb install -r app-release.apk` (a debug build must be uninstalled first, the signatures differ).
