# Kanban board for undated tasks

Status: approved by the owner on 2026-11-16 as one slice, with full autonomy ("spec, design, review, criticise, implement, test, deploy"). Inline execution, no sub-agents. The review and critique are recorded at the end and are already applied in the text above them.

## Intent

Some tasks have no date: "Call plumber". They do not belong on the calendar, and chat gives them no overview. The owner wants a third tab, a Kanban board, that shows these tasks in three columns: **To pick up**, **In progress** and **Done**.

## Goals

- A **Board** tab beside Chat and Calendar shows every task that has no date, grouped by status.
- Moving a task between columns takes one tap and leaves no noise in chat.
- Tasks created in chat appear on the board. The owner can also add a task on the board directly.
- Chat can move a task too ("start the plumber task").

## Non-goals and anti-goals

- No drag and drop, swimlanes, custom columns, WIP limits, manual card order, subtasks or labels.
- No board for dated tasks. The calendar owns them. The board does not duplicate them.
- The board must not make the owner confirm each move. A move that needs a preview and Confirm has failed.
- A move must not add preview or result cards to chat.
- A moved task that keeps firing reminders, or a done task that still nags, has failed.

## Approaches

1. **Board of undated tasks, status as the column** (chosen). Every item already has `status` (`open`, `in_progress`, `done`, `cancelled`). The columns are those statuses. A dated task stays on the calendar. Every live item then shows in exactly one place: the calendar (it has a date) or the board (it has none). No schema change and no second source of truth.
2. A separate `board_column` field on items. Rejected: it duplicates `status`, so a task could be done in one place and open in another, and it needs a migration.
3. Show all tasks on the board, dated or not. Rejected: the owner scoped the board to tasks without a date, and the calendar already owns dated work.
4. Move through the preview and Confirm flow like other changes. Rejected: three taps and two chat cards per move. Reminder Done and Snooze already act without a preview; a move is as small and as reversible.

## Design

### Which items are on the board

`domain.OnBoard(item)` is true when all hold:

- `kind` is `task` or `reminder` (not `event`);
- no `start_at`, `due_at`, `start_date` or `due_date`;
- no recurrence (a recurring item needs an anchor date);
- `status` is `open`, `in_progress` or `done`.

`remind_at` does not count as a date. "Call plumber, remind me Friday" stays on the board with its reminder.
`cancelled` items are on neither screen. A done task shows only while it is recent: completed (or, when `completed_at` is missing, last updated) in the last 14 days, newest 30. Older done tasks stay in the database and chat history.

If a task gains a date (chat: "call plumber on Friday"), it leaves the board and appears on the calendar. If it loses its date it returns to the board. The board needs no other rule.

### Server

- `domain.OnBoard`, `domain.BoardColumns(items, now)` give three lists, sorted: To pick up and In progress by priority (1 highest, unset last), then oldest `created_at` first; Done by `completed_at`, newest first.
- `GET /v1/board` returns `{"open":[...],"in_progress":[...],"done":[...]}` using the item JSON of `GET /v1/items`.
- `POST /v1/items/{id}/status` with `{"expected_version":n,"status":"open|in_progress|done"}` changes the status in one transaction: item, audit row (actor `owner`) and reminder schedule. No chat entry. It returns the updated item. A wrong version returns 409 `conflict`. An item that is not on the board returns 400 (`This task is on the calendar.`).
  - To `done`: set `completed_at` to now. From `done` to another status: clear `completed_at`.
  - Moving to `done` closes the reminder schedule and nags (the existing reminder projection does this for non-open items). Moving back reopens the schedule only for a future reminder.
- `POST /v1/items` with `{"title":"..."}` creates an open task: kind `task`, default timezone, version 1, audit row (actor `owner`), no chat entry. Validation is `domain.Validate`; the title is trimmed, 1 to 200 characters. Returns 201 and the item.
- Both writes use the existing owner authentication and same-origin check.
- Logs: `board loaded` (counts), `board item moved` (`item_id`, `from`, `to`, `version`), `board item added` (`item_id`). No titles.
- Agent: the proposal gets an optional `status` (`open` or `in_progress`) for `create_item` and `update_item`. `propose_complete` still marks done; an update with `status: open` reopens a done task. The prompt says: undated tasks appear on the Board; do not invent a due date for them; use `status` when the owner starts or pauses a task.

### Client

- `NavDestination.board` joins the bar: Chat, Board, Calendar. The root route stays Chat. Board and Calendar replace each other, so the stack never grows beyond two routes.
- `BoardController` (application layer) loads the board, moves tasks, adds tasks, and exposes `busy` item IDs, `error` and the last move for Undo. It talks to a `BoardRepository` interface that `ChatApi` implements.
- **Layout.** Width under 720 px: a segmented header with the three column names and counts, one column visible, swipe or tap to switch. At 720 px and wider: three columns side by side. A column is a scrolling list with pull to refresh.
- **Card.** Left colour stripe (the item colour), title (up to three lines), priority mark when set, list name when set, a bell with the reminder time when set, and one 44 px text button: To pick up → **Start**, In progress → **Done**, Done → **Reopen**.
- **Details sheet.** Tap a card. It shows the title, notes (the chat Markdown renderer), list, reminder, contact and booking fields, and the moves to the other two columns. It also has **Delete**, which reuses the calendar's delete preview and Confirm dialog.
- **Add.** The To pick up column starts with an "Add a task" field. Submit creates the task and clears the field. The field disables while the request runs.
- **Undo.** After a move a snackbar says `Moved to <column>` with **Undo**. Undo is a move back using the new version.
- **States.** Loading shows a thin progress line, not a blocking spinner. An error shows a message with a Retry button in the content. Empty columns say what belongs there: `Nothing to pick up. Add a task above or tell Minder.`, `Nothing in progress.`, `Nothing done in the last 14 days.`
- The board loads when the tab opens and after each of its own actions. Tasks Minder creates in chat show up the next time the tab opens or the owner pulls to refresh.
- Style follows `foundation.md`: Tokens, square controls, orbital background, the screen header `BOARD`, Material icons plus the Carbon glyphs the bar already draws.

## Constraints

- No schema migration; keep old migrations unchanged.
- Business rules (board membership, sorting, transitions) live on the server. The client only displays and sends.
- Logs carry IDs and counts, not titles.
- Version: app 0.14.0+39 (split arm64 code 2039). Production must run the matching server before the app is delivered.

## Acceptance

1. Server tests on real temporary SQLite: membership rule (dated, evented, recurring, cancelled and undated items), column order, 14-day and 30-item done window, status moves (all six transitions between the three statuses), `completed_at` set and cleared, version conflict, no chat entry written, audit row written, reminders closed on done and reopened for a future reminder, create task, quick-add validation, `not on board` rejection, private values absent from logs.
2. Agent tests: `status` in the tool schema; proposal applies it; prompt mentions the board and no invented due date.
3. API tests for the three routes, including 400 and 409.
4. Flutter tests: controller (load, move, undo, add, conflict, error and retry), wire parsing, board widget in narrow and wide layouts, card button per column, details sheet, bottom bar with three items and navigation between Chat, Board and Calendar, golden for the narrow board.
5. `toolchain/check` exits 0. A new browser scenario on release web: add a task, start it, finish it, undo, reopen, confirm chat has no new cards. Existing chat, calendar and email scenarios still pass.
6. Production runs the new server; the APK is built, signed with the cached key and served on the LAN; the handoff records evidence.

## Left to the builder

Pixel spacing within the tokens, glyph drawing for the Board icon, exact snackbar wording beyond the text above, helper names.

## Hardening after review

After the first release a review found weak points and they were fixed: the agent no longer supplies a version and its chat history carries no IDs; every item write goes through one `saveItem` helper; a preview says when an edit moves a task between Board and Calendar and the calendar tags in-progress cards; the board read is narrowed in SQL (no index, to keep older servers able to open the database); the board refreshes when chat gets a new entry; the Board API client and the delete interface are separate from `ChatApi`.

## Known limits

- A dated task has no board card, so it cannot be "in progress" visibly. Its status still exists and chat can change it.
- Done tasks older than 14 days are not shown anywhere except chat history.
- Refresh follows chat entries only; a change made on another device appears on the next load.
- SQLite still scans the item table for the board; an index needs a schema 7 migration that would stop an older server opening the database.
- Card order is fixed by priority and age; the owner cannot reorder.

## Review and critique (applied above)

Findings from reading the spec against the code and the owner's intent:

1. *Dated tasks that vanish.* A task in progress that gets a date leaves the board. Kept on purpose, because it gives every live item one home. The limit is recorded and chat says nothing misleading: the item still exists on the calendar. Accepted.
2. *Move via Preview and Confirm.* Rejected in Approaches, because it fills chat with cards. A direct move needs its own safe path: version check, audit row, and the item must be on the board, so the endpoint cannot alter calendar items by accident. Added.
3. *Chat Undo noise.* The existing Undo writes an "Undone" chat entry. A board Undo is a reverse move instead, so chat stays quiet. Added.
4. *Done pile.* Without a window the Done column grows forever and slows the page. Added the 14-day window and 30-item cap.
5. *Reminders.* A done task with an active nag would keep notifying. The move runs the same reminder projection as other item writes, and tests assert closed and reopened schedules. A reopened task with a past `remind_at` does not fire again. Added to acceptance.
6. *`remind_at` as a date.* Reading "not tied to a date" strictly would drop reminder tasks from both screens, because the calendar ignores `remind_at`. Resolved: a reminder is not a date.
7. *Three-button cards* crowd a 360 px phone. Reduced to one forward button per card; other moves live in the details sheet.
8. *Drag and drop.* Cut. It fails on the phone with three off-screen columns, keyboards and screen readers. Buttons are one path that works everywhere.
9. *Three tabs and the route stack.* Chat is root; Board and Calendar replace each other, so Back always returns to Chat and the stack stays shallow. Added.
10. *Agent drift.* Without a `status` field the agent cannot start a task, and without a prompt rule it tends to add due dates that push tasks off the board. Added the field and the rule.
11. *Double submit.* Quick add and moves disable their control while the request runs, and the server's version check rejects a repeated move.
12. *Concurrency with chat.* A 409 reloads the board and shows `This task changed. The board is updated.` No silent overwrite.
13. *Security.* New writes sit behind owner authentication and the same-origin check. The title is plain text and renders without HTML.
14. *Scale.* The server loads all items and filters in Go, like `ItemsBetween`. Fine for one owner; no index needed.
