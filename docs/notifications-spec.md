# Reminder and notification slice

The owner approved work on 2026-11-11 after accepting the M0 phone build.
Implement inline. Keep changes isolated from main. Deliver an updated APK through the existing Telegram bot.

## Outcome

Chat can propose an explicit reminder time and nag interval. Confirm saves it.
A reminder appears in chat when due. Android receives Firebase data messages with Done, Snooze and Open actions.
Swiping does not acknowledge. Done completes the item. Snooze postpones its reminder. Open acknowledges the current reminder.
Notification actions open the authenticated app before changing server data. No public action endpoint or embedded server credential exists.

## Scheduling

`remind_at` is independent from `due_at`. Existing items do not acquire reminders automatically.
`nag_minutes` ranges from 1 to 1440; zero means one alert without repeated nags.
An explicit reminder defaults to 15-minute nagging when requested through chat.
The database stores each reminder generation and next firing time. Schedule changes invalidate old action IDs.
A worker polls every 15 seconds. Each occurrence enters chat once and records a durable push delivery per device.
Failed pushes retry after one minute. Successful sends are provider acceptance, not proof of phone delivery.
A restart catches up overdue reminders. It emits one late alert per reminder, then resumes nagging from the current time.
Done, Open and Snooze are idempotent for their occurrence ID. Old actions cannot alter a newer occurrence.
Completion, deletion and clearing a reminder cancel future delivery. Undo restores the applicable schedule.

## Push and UI

Use FCM HTTP v1 and high-priority data messages. Display notifications in native Android code for closed-app support.
Use a high-importance notification channel, the monochrome Minder badge, private lock-screen visibility and three actions.
Request Android notification permission through an explicit Enable notifications button. Show configuration/permission failures.
Firebase public Android configuration comes from the authenticated server. Persist it for background process initialization.
Service-account JSON stays in a runtime environment variable. No credential or device token enters logs.
Token changes register through the authenticated API. Pending deliveries cover newly registered devices.
The web client shows the same reminder cards and actions. Browser push is outside this slice.

## Boundaries

This slice supplies the reminder path required for notifications. General lists, editable profiles and recurrence remain later M1 work.
Advanced escalation, ringing, and custom snooze times remain later M2 work. Initial snooze choices: 15 minutes, one hour, tonight, tomorrow.
Defaults use Europe/Amsterdam, including daylight-saving changes for tonight and tomorrow.
Firebase project setup and real push delivery require the owner's Firebase configuration. Continue all local work while it is unavailable.

## Acceptance

- No reminder before Confirm, or for old items with only a due time.
- Restart catch-up, nag intervals, retries, multiple devices and invalid tokens preserve durable state.
- Completing, clearing, rescheduling, acknowledgement and Undo have transaction tests.
- Duplicate and stale notification actions cannot change a newer reminder.
- FCM boundary tests check authorization, data, priority, invalid-token classification and transient failure.
- Flutter web exercises a due reminder and action against local Go and SQLite.
- Android build and emulator checks verify permission and notification/action rendering.
- Real Firebase delivery and phone approval remain explicitly pending until tested.
