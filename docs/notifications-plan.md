# Reminder and notification implementation plan

Use executing-plans inline, with test-first behavior changes and one relevant final verification pass.
Spec: [notifications-spec.md](notifications-spec.md).

## Tasks

- [x] Add `RemindAt`, `NagMinutes` and clear flags to item/proposal validation, preview and model tools.
  Files: `server/domain/model.go`, `server/command/command.go`, `server/assistant/{model,assistant}.go`.
  Tests prove explicit opt-in, invalid values and preview/Confirm behavior.
- [x] Add durable scheduling and delivery rows in `server/store/reminders.go` and schema version 2.
  Interfaces: `DueReminders(now time.Time)`, `ReminderAction(id, action string, now time.Time)`, `RegisterDevice(id, token string)`.
  Item writes synchronize schedules within their existing transaction. Worker ticks create chat entries and per-device delivery rows.
  Tests cover restart, catch-up, deduplication, retries, state changes and stale actions with real SQLite.
- [x] Add `server/reminder` worker and FCM HTTP v1 sender.
  `Tick(ctx, now)` processes bounded batches. Sender accepts token and string data, returns an invalid-token classification.
  Tests use an HTTP boundary and fake clock, never real owner tasks.
- [x] Add authenticated reminder/device/config routes in `server/api/reminders.go`, wire worker in `cmd/minder/main.go`.
  GET reminders and push config; POST device registration and reminder actions. Strict bounded JSON and route validation.
- [x] Add Flutter reminder cards and notification setup in `app/lib/{domain,data,ui}`.
  DTOs carry reminder occurrence IDs. Actions use the existing authenticated repository and Retry flow.
  Unit/widget tests prove action routing and error visibility.
- [x] Add native Android FCM service, persistent configuration and permission/token bridge.
  Notification actions launch the app, which sends authenticated actions. An emulator-only bridge probe tests display without Firebase.
- [x] Run Go race tests/vet, Flutter tests/analyze, one local web reminder scenario, and Android build/emulator checks.
  Package APK only with accurate Firebase setup status. Deploy/send within the owner's authorization when runtime setup is ready.
  Record exact evidence and remaining gaps. Preserve the branch until phone approval.

## Review focus

Old due dates must not create surprise alerts. Schedule edits invalidate pending deliveries and action IDs.
Failures leave retryable deliveries. No token or service-account content is logged or served publicly.
A denied notification permission stays visible. App startup without Firebase still supports chat.

## Evidence (2026-11-11)

- `go test -race ./...` and `go vet ./...` pass. `flutter test` (12 tests) and `flutter analyze` pass.
- `toolchain/reminder-local-check` passes: a confirmed reminder reaches chat with Done, Snooze and Open, and Done handles it. Screenshot: `evidence/local-reminder.png`.
- The release web build and the arm64 APK build.
- Not run: Android emulator check (no emulator container), real FCM delivery, phone approval.
- Blocked: the server needs the Firebase service-account JSON in TokenCrypt. `firebase_minder_key` is not that file.
