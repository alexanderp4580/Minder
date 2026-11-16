# Kanban board

Date: 2026-11-16. App 0.14.1+40 (0.14.0+39 was the first release of the same feature). Spec: [kanban board design](../superpowers/specs/2026-11-16-kanban-board-design.md). Plan: [kanban board plan](../superpowers/plans/2026-11-16-kanban-board.md).

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
| Formal code review (`/code-review high` on the branch) | 10 findings; 9 fixed with a failing test first (agent reopen kept a past reminder; stale `load()` overwrote a newer board; refusal text for cancelled tasks; a failed move did not reload; `Board` read sources per item; duplicate JSON decoder; creation times sorted as text; sheet moved a stale version; tab index after wide/narrow switch). One kept on purpose: `update_item` may reopen a cancelled item because the owner sees and confirms the preview |
| Browser scenarios on release web | `board-local-check`, `calendar-local-check`, `reminder-local-check`, `email-local-check` and `local-check` all printed PASS |

## Real model

`toolchain/real-model-board-check` (chat only, temporary database, Haiku 5.5): with the first prompt rule 1 of 6 runs passed. Traces showed the model copying a change ID from chat history as an item ID, guessing the version, and listing tasks instead of proposing an update. After the prompt rule named `propose_update`, `status` and "IDs only from Current items", 8 of 8 runs passed (3 model calls, about 0.0058 USD each). The preparation real-model check passed (four items, timing options, task scheduled three days before). `real-model-multi-check.py` fails identically on the pre-board server (`2a2633f`): its fixture expects `due_at` fields, but the agent now moves appointments with `start_at`. Not changed here.

## Not verified

- No phone check of the Board, Undo snackbar or navigation.
- No live update: tasks created in chat appear when the Board tab opens or on pull to refresh.
