# Notification action evidence

Date: 2026-11-12. Branch: `notification-body-tap`.

## Change

- Tapping a reminder notification body does nothing. The notification has no content intent.
- Done and Snooze run in the background. They no longer open the app.
- Open still acknowledges the reminder and opens the app.
- A failed background action keeps the notification, clears the stored action and shows a message.

## Checks

| Command or check | Observed result |
|---|---|
| `background_action_test.dart` before the code | Failed: the function did not exist |
| Same test after | Passed: five cases covering success, a 409 reply, a network error, no stored action and an invalid action |
| `toolchain/check` | Passed, exit 0 |
| Emulator `emulator-5584`, debug x86_64 build 0.4.8, installed over 0.1.1 | The app process was cold. A Snooze broadcast reached `ReminderActionReceiver`. The local server logged `reminder action` with `action=snooze15` and the next time 15 minutes later |
| Same run, foreground app | The launcher stayed on top. The app never came forward |
| Same run, stored state | `handled_<id>` is true and no pending action remains |
| Same run, server stopped, Done broadcast | No crash. No pending action remains. The toast was suppressed, because notifications are off for the app on this emulator |
| Release arm64 APK | Built and signed with the earlier key |

The emulator check used `adb root` to send the broadcast, a local Go server with the model fixture and `adb reverse`.
Debug builds now allow plain HTTP to localhost for this purpose. Release builds do not.

## Limits

- No physical phone. Real notification buttons, the Android 12 and later background limits, battery restrictions and the toast on a phone with notifications on are unchecked.
- The test sent the broadcast directly. It did not press a rendered notification button, because Firebase delivery needs the production project.
- Cancelling the notification after success uses the code the app already used. The run did not show a notification.
