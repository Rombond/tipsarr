# Tipsarr Android

Kotlin + Jetpack Compose + Material 3 (minSdk 26, compileSdk 37). It follows the iOS app: same behaviour, same screens, similar UI (`../ios/`). Same API (`../api/openapi.yaml`, copied by `make generate` to `app/src/main/openapi/`), same tokens (`../design/tokens.json`, generated to `app/.../design/Tokens.kt`), same strings (`app/src/main/res/values{,-fr}/strings.xml`, from `../design/strings`). Icon layers come from `../design/icon/Android-*`.

Build and run (Android Studio's bundled JDK, SDK in `~/Library/Android/sdk`):

```sh
export JAVA_HOME="/Applications/Android Studio.app/Contents/jbr/Contents/Home"
export ANDROID_HOME=$HOME/Library/Android/sdk
echo "sdk.dir=$ANDROID_HOME" > local.properties     # once, not committed
./gradlew installDebug                               # with an emulator or phone connected
```

`make generate` refreshes the generated files. Do not edit `Tokens.kt`, `strings.xml` or `openapi.yaml` by hand.
