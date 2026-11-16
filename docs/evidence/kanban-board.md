# Kanban board

Date: 2026-11-16. App 0.14.0+39. Spec: [kanban board design](../superpowers/specs/2026-11-16-kanban-board-design.md). Plan: [kanban board plan](../superpowers/plans/2026-11-16-kanban-board.md).

## Behaviour

A Board tab (Chat, Board, Calendar) lists tasks without a date in To pick up, In progress and Done. Moves and quick adds write an owner audit row and no chat entry. The agent can set `status` (`open`, `in_progress`) and is told not to invent due dates for undated tasks.

## Checks

| Check | Result |
|---|---|
| Domain, command, API, tool schema and prompt tests (Go, real temporary SQLite) | Pass under `toolchain/check` (race, vet, gofmt) |
| Review findings fixed with a failing test first | Reopening a done task with a past reminder re-notified at once (now cleared); Undo offered an Undo of itself; the Undo snackbar was unreadable (dark theme added); the bar overflowed at 1.4× text (items grow) |
| Flutter tests | 228 passed; analyze clean; format clean |
| Goldens | `board.png` added; `calendar.png`, `calendar-details.png` and `m0-chat.png` regenerated for the third bar item and inspected |
| `toolchain/check` | Exit 0 |
| Browser scenarios on release web | `board-local-check`, `calendar-local-check`, `reminder-local-check`, `email-local-check` and `local-check` all printed PASS |

## Not verified

- No real-model run of the prompt change: the OpenRouter grant was not active and no approval was requested. The change adds one rule and one optional field.
- No phone check of the Board, Undo snackbar or navigation.
- No live update: tasks created in chat appear when the Board tab opens or on pull to refresh.
