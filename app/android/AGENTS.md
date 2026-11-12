# Purpose

Android platform integration.

# Ownership

Gradle configuration, manifest, Kotlin activities, FCM integration and generated launcher assets.

# Local Contracts

Application ID is `io.liveleds.minder`. Callback uses the auth management activity and empty callback task affinity.
The auth manager uses singleTask. Firefox callbacks must return their result to Minder.
Keep native notification method payloads compatible with data/notification_device.dart.
A reminder notification has no content intent: tapping its body does not open the app. Only Open brings the app forward.
Done and Snooze send a broadcast to `ReminderActionReceiver`. `BackgroundAction` starts a headless engine at the Dart entry point `notificationAction`, with a nine-second deadline. The app screen and the engine share `MinderNotifications.handleAction`.
Debug builds allow plain HTTP to localhost, 127.0.0.1 and 10.0.2.2 for emulator checks (`src/debug`).
Generate launcher icons from brand/app-icon.svg; do not hand-edit generated icon files.

The minder/attachments channel opens authenticated downloads from a narrowly scoped cache FileProvider and reports the device timezone. Capture review notifications use generic copy and open chat; reminder body/button behaviour stays unchanged.

# Work Guidance

Preserve the cached debug signing key for installable updates. Native checks use disposable emulators only.

# Verification

`toolchain/android-callback-check.py` verifies callback ownership and delivery on a disposable emulator.
The owner authorized orbital preview APK builds and Telegram delivery. Phone approval still gates merging.

# Child DOX Index

None.
