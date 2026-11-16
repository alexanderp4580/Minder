# Kanban board plan

Governing spec: [kanban-board-design](../specs/2026-11-16-kanban-board-design.md). Inline execution; test first for every task.

## Task order and interfaces

1. **Domain.** `OnBoard(Item) bool`; `Board{Open, InProgress, Done []Item}`; `BoardColumns([]Item, time.Time) Board` with the sorting and done window from the spec; `Proposal.Status`.
   Tests: membership table, sort order, done window and cap.
2. **Command.** `Service.MoveTask(id string, version int, status string) (domain.Item, error)` and `Service.AddTask(title string) (domain.Item, error)`. Both run in one transaction, write an audit row with actor `owner` and no chat entry. `update_item` and `create_item` apply `Proposal.Status` (`open`, `in_progress`).
   Tests on real SQLite: six transitions, `completed_at`, conflict, not-on-board, reminder schedule closed and reopened, no chat entries, validation, logs without titles.
3. **Store and API.** `Store.Board(now)`; `GET /v1/board`, `POST /v1/items/{id}/status`, `POST /v1/items`. Interfaces `Queries.Board`, `Commands.MoveTask`, `Commands.AddTask`.
   Tests: status codes 200, 201, 400, 409, auth and same-origin, JSON shape.
4. **Agent.** `status` in the proposal tool schema; prompt rule on the Board and on not inventing due dates.
   Tests: schema and prompt assertions; proposal applies the status.
5. **Client data and application.** `BoardRepository`, `Board` value, `ChatApi` methods, wire parsing, `BoardController`.
   Tests: controller and adapter with a fake HTTP client.
6. **Client UI.** `BoardScreen`, card, details sheet, add field, `NavDestination.board`, three-item `BottomNav`, navigation in `main.dart`, Board glyph.
   Tests: widget tests (narrow and wide), bottom bar, navigation, golden.
7. **Docs and DOX.** Child AGENTS.md files, architecture, screens, requirements, README routes, handoff.
8. **Verify.** `toolchain/check`, new `board-local-check` browser scenario, existing scenarios, review, fix.
9. **Release.** Version 0.14.0+39, APK build, server deploy through Coolify, LAN link, evidence.
