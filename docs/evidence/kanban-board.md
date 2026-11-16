# Kanban board

Date: 2026-11-16. App 0.14.2+41 (0.14.0+39 and 0.14.1+40 were earlier releases of the same feature). Spec: [kanban board design](../superpowers/specs/2026-11-16-kanban-board-design.md). Plan: [kanban board plan](../superpowers/plans/2026-11-16-kanban-board.md).

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

## Hardening (0.14.2)

Each item below got its test first: version fill for agent proposals, history without IDs, `shown_on` preview field, SQL read equal to the Go filter (property test), `refresh` and `BoardChatSync`, `In progress` calendar tag. Real model after the change: board check 8 of 8, multi-proposal check 4 of 4. `toolchain/check` exit 0 (237 Flutter tests) and all five browser scenarios passed.

## Model credit

The preparation real-model check passed once and then failed twice on this code; the pre-board server failed it 3 of 3 with a logged `402`. The OpenRouter account showed `total_credits` 25 against `total_usage` 25.01, so model calls fail until credit is added. Those failures are not attributed to the code. Rerun `toolchain/real-model-agent-check --preparation` after a top-up.

## Prompt caching

Tests: the stable prompt prefix holds nothing that changes per call; requests mark the system prefix and the turn's own message and leave other messages plain strings; `toolchain/check` exit 0 (237 Flutter tests) and the chat, calendar, email and reminder scenarios pass with marked requests. Real model (3 runs of `real-model-board-check`): prompt 17,025 to 17,677 tokens, 16,533 cached from the second call on, cost per run $0.0032 for the cold run and $0.0013 after (was $0.0056).

## Not verified

- No phone check of the Board, Undo snackbar or navigation.
- No live update: tasks created in chat appear when the Board tab opens or on pull to refresh.
