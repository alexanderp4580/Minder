# Architecture cleanup evidence

Date: 2026-11-12. Branch: `m0-chat`. Starting commit: `a330733`.
Spec: [../superpowers/specs/2026-11-12-architecture-cleanup.md](../superpowers/specs/2026-11-12-architecture-cleanup.md).
Plan: [../superpowers/plans/2026-11-12-architecture-cleanup.md](../superpowers/plans/2026-11-12-architecture-cleanup.md).

## Completed scope

All six approved steps are complete. Work ran inline in the existing worktree.

| Step | Result |
|---|---|
| Agent0ai DOX | Root framework, MIT notice, 34 scoped contracts, direct child indexes and executable checks |
| Go persistence boundaries | Application services use repository interfaces and `port.Transaction`; only SQLite executes SQL |
| Go business policy | Domain owns snooze/projection rules and profile defaults; reminder services own actions, scheduling and retries |
| External adapters and requests | OpenRouter, FCM and authentication are adapters; conversation owns acceptance, concurrency and reply lifetime |
| Flutter cleanup | Controllers use domain repositories directly; data owns transport; ProviderScope owns shared clients |
| Review and verification | Dependency/DOX guards, rollback and lifecycle regression tests, full suites and compiled browser scenarios |

No dependency versions, SQL migrations, HTTP routes, JSON field names or golden images changed.
API queries use a read interface directly; they do not need a forwarding query class.
SQLite maintains derived reminder projections in the same transaction as item/profile writes.
Domain functions define those projection rules. The adapter contains no snooze choices or reminder action orchestration.

## Reproduced defects and coverage

- A storage failure during Confirm surfaced as a version conflict. It now retains its error cause and pending state.
- Chat disposal closed a repository shared by other controllers. ProviderScope now owns shared API/auth client cleanup.
- Late native notification results could register devices or apply actions after controller disposal. Disposed controllers stop that work.
- The first dependency checker missed conditional Dart imports and nested Go network imports. Negative fixtures now cover both.

New transaction tests use real temporary SQLite databases.
They verify rollback of item, event and audit writes, and rollback of a failed reminder result.
A successful retry returns the recorded result and creates one audit entry.
Existing tests cover profile schedule changes, catch-up, nagging, recurrence, stale actions and Undo.

Conversation tests verify busy rejection, cancellation, safe errors and waiting for accepted replies before storage cleanup.
The process lifecycle test verifies that shutdown waits for the worker.
Existing HTTP tests cover auth gates, malformed requests, streams, conflicts, branding and assistant panic recovery.

## Checks run

| Command | Observed result |
|---|---|
| `toolchain/check` | Passed, exit 0 |
| Structure checker unit tests | 7 passed, including forbidden imports, conditional imports, missing owners and missing indexes |
| Git hook tests | 7 passed |
| `toolchain/architecture-check` | Passed, no Go or Flutter dependency violations |
| `toolchain/dox-check` | Passed, owners, section order, child indexes and local links |
| `toolchain/run go go test -race ./...` | Passed for all packages with tests |
| `toolchain/run go go vet ./...` | Passed |
| `toolchain/run go gofmt -l .` | No output |
| `toolchain/run flutter flutter test` | 32 passed, including unchanged chat/calendar goldens |
| `toolchain/run flutter flutter analyze` | No issues |
| `toolchain/run flutter dart format --set-exit-if-changed lib test` | Zero changes |
| `toolchain/run flutter flutter build web --release --no-wasm-dry-run` | Passed |
| `toolchain/calendar-local-check` | Passed, placement and back navigation |
| `toolchain/reminder-local-check` | Passed, reminder card and Done |
| `toolchain/local-check` with the fixture | Passed, preview, Confirm, query, Discard, change, Done, Undo, reload, offline retry and SQLite persistence |
| Real-model multi-change check, three runs | Passed, two previews with the expected times in all three runs |
| `git diff --check` | Passed |

The web build retains the SDK's Cupertino font-family warning. The application uses Material icons.
The pinned Flutter container also prints its root-user warning and newer-version notices.
These warnings did not fail the build or rendering checks.
Local logs and this run's browser screenshots remain under `.local/`; credentials did not enter the evidence.

## Final review

Review ran inline against the approved design and current source.
It checked package/layer dependencies, transaction atomicity, schedule generations, retries, error propagation and resource ownership.
It also checked schema/wire compatibility, DOX coverage, direct child indexes and documentation links.
The identified architecture and lifecycle defects are resolved. No approved cleanup task remains open.

## Release and product boundaries

This work did not deploy, push, merge or build APK 0.4.0. Phone approval still gates merging.
The earlier verified Coolify identity variables remain ready for the next deployment.
Owner phone tests, date-only interpretation, weaker-model omissions and the icon-guide difference remain product follow-ups in the handoff.
The refactor preserves those product behaviours; its tests do not claim to resolve them.
