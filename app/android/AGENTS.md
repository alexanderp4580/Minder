# Purpose

Android platform integration.

# Ownership

Gradle configuration, manifest, Kotlin activities, FCM integration and generated launcher assets.

# Local Contracts

Application ID is `io.liveleds.minder`. Callback uses the auth management activity and empty callback task affinity.
The auth manager uses singleTask. Firefox callbacks must return their result to Minder.
Keep native notification method payloads compatible with data/notification_device.dart.
Generate launcher icons from brand/app-icon.svg; do not hand-edit generated icon files.

# Work Guidance

Preserve the cached debug signing key for installable updates. Native checks use disposable emulators only.

# Verification

`toolchain/android-callback-check.py` verifies callback ownership and delivery on a disposable emulator.
The owner authorized orbital preview APK builds and Telegram delivery. Phone approval still gates merging.

# Child DOX Index

None.
