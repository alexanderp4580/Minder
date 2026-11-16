# Purpose

State and lifecycle orchestration.

# Ownership

ChatController, CalendarController, BoardController, NotificationController and `performNotificationAction`, which runs a stored notification action without a screen.

# Local Contracts

Use domain repository/device interfaces directly. Keep transport maps and routes out.
Flutter foundation supplies ChangeNotifier; UI widgets and data implementations do not belong here.
Calendar requests ignore superseded results. Notification state reflects registration and permission.
Disposed notification controllers stop work after pending native results and leave unhandled actions for retry.
Controllers cancel their own work; they do not close injected shared repositories.

ChatController owns review replies from the main composer and shortcut buttons. It uses the newest open question's source, revision and candidate context through CaptureRepository; it never parses answer text. Shared optimistic keys hide both copies of answered choices, and later owner entries preserve suppression after history reloads. A refused send restores choices and custom text. A newer capture revision completes only its own pending review wait; unrelated capture updates do not end normal chat waiting.

ChatController keeps the latest rendered `cursor` separate from its history/event-stream watermark. Direct action, answer and send responses update visible entries without advancing that watermark; polling and reconnects must still read earlier transaction events. `thinking` is true while normal chat waits, while the owner's message is the newest entry (a quick reply or reply still queued, also after a reconnect), or while a canonical email card has status received/processing. `send` refuses text over `maxMessageBytes` (400,000 UTF-8 bytes) with a visible message and returns it to the composer; nothing is cut.

ChatController marks `showOffline` only after the connection stays down for the grace period (2 seconds); transport failures (`RepositoryFailure.transport`) never set `error`: only the header icon shows them. A failed send keeps its text in `takeUnsentText()` and the screen puts it back in the composer. An exception the data layer did not wrap (a reset event stream) counts as a dropped connection. Other failures set `error`, a message without Retry. A message from your own send or action clears on the next send or action. A message from a background sync or the event stream clears when the connection works again, and so does a sign-in message. A `removed_entry_ids` list on a result removes those entries from memory. CalendarController previews, confirms and discards item deletes through ItemRepository. CaptureController owns targeted and quick-reply answers (the server queues them as owner messages for the agent and returns the owner entry, so they appear as the owner's own bubble), dismissal, source actions and disposal guards. Capture events never clear unrelated chat waiting. Calendar selection includes occupied event/date ranges.

BoardController loads the board and sends moves and new tasks through `BoardRepository`; the server owns every rule. A move marks the task busy, reloads the board and keeps the finished move (`lastMove`) so the screen can offer Undo, which moves the task back with its new version. A 409 (`RepositoryFailure.conflict`) reloads the board and says `This task changed. The board is updated.` A failed move or add keeps the old board and shows the server's message. Requests superseded by a newer load or an action's reload are ignored, so a slow older load cannot restore a stale board. Any server answer to a failed move reloads the board; a dropped connection does not. `refresh()` reloads silently (no loading bar, no error) and only after a first load. `BoardChatSync` calls it when chat's cursor changes, so tasks the agent creates or moves appear without reopening the tab. `ItemDeleter` is the delete interface shared with CalendarController.

# Work Guidance

Keep responsibilities focused. Preserve existing widget appearance during refactors.

# Verification

From root: `toolchain/run flutter flutter test`, `toolchain/run flutter flutter analyze`, `toolchain/architecture-check`.

# Child DOX Index

None.
