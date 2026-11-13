# Purpose

State and lifecycle orchestration.

# Ownership

ChatController, CalendarController, NotificationController and `performNotificationAction`, which runs a stored notification action without a screen.

# Local Contracts

Use domain repository/device interfaces directly. Keep transport maps and routes out.
Flutter foundation supplies ChangeNotifier; UI widgets and data implementations do not belong here.
Calendar requests ignore superseded results. Notification state reflects registration and permission.
Disposed notification controllers stop work after pending native results and leave unhandled actions for retry.
Controllers cancel their own work; they do not close injected shared repositories.

ChatController marks `showOffline` only after the connection stays down for the grace period (2 seconds); transport failures (`RepositoryFailure.transport`) never set `error`: only the header icon shows them. A failed send keeps its text in `takeUnsentText()` and the screen puts it back in the composer. An exception the data layer did not wrap (a reset event stream) counts as a dropped connection. Other failures set `error`, a message without Retry. A message from your own send or action clears on the next send or action. A message from a background sync or the event stream clears when the connection works again, and so does a sign-in message. A `removed_entry_ids` list on a result removes those entries from memory. CalendarController previews, confirms and discards item deletes through ItemRepository. CaptureController owns targeted and quick-reply answers, dismissal, source actions and disposal guards. Capture events never clear unrelated chat waiting. Calendar selection includes occupied event/date ranges.

# Work Guidance

Keep responsibilities focused. Preserve existing widget appearance during refactors.

# Verification

From root: `toolchain/run flutter flutter test`, `toolchain/run flutter flutter analyze`, `toolchain/architecture-check`.

# Child DOX Index

None.
