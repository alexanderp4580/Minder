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

# Work Guidance

Keep responsibilities focused. Preserve existing widget appearance during refactors.

# Verification

From root: `toolchain/run flutter flutter test`, `toolchain/run flutter flutter analyze`, `toolchain/architecture-check`.

# Child DOX Index

None.
