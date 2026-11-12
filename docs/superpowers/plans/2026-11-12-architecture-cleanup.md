# Architecture cleanup implementation plan

> Execute inline with `executing-plans`; the owner approved continuous work without sub-agents.

**Goal:** Finish the architecture cleanup and Agent0ai DOX tree.
**Spec:** [../../superpowers/specs/2026-11-12-architecture-cleanup.md](../../superpowers/specs/2026-11-12-architecture-cleanup.md).
**Architecture:** Domain and application contracts face inward. SQLite and provider adapters implement those contracts.
**Tech stack:** Existing Go, SQLite, Flutter and Podman toolchain.

## Global constraints

Preserve product behaviour, routes, schemas, cost limits and UI. No new dependencies. No merge, deploy or APK build.
Reuse `m0-chat`; work inline. Keep credentials in memory and runtime variables.

## Tasks

- [x] 1. Install the root DOX framework and dependency checks. Prove the checks reject current infrastructure coupling.
- [x] 2. Add `port.Transaction` and narrow service repositories. Adapt SQLite transactions; move identifiers into domain.
      Verify rollback, durable events, conflict handling and preview batches with the existing real-SQLite tests.
- [x] 3. Move reminder policy/actions/scheduling to `domain` and `reminder`. Keep atomic projections in storage.
      Verify nag/catch-up, snooze variants, stale actions, recurrence, idempotency and delivery state.
- [x] 4. Extract OpenRouter and FCM adapters. Move message acceptance and reply lifecycle into `conversation`.
      Make HTTP handlers depend on interfaces. Verify provider payloads, error redaction, panic recovery and busy rejection.
- [x] 5. Remove Flutter forwarding classes and use domain repositories in controllers. Preserve existing goldens and native payloads.
      Fix lifecycle defects with regression tests when reproduced.
- [x] 6. Complete all child DOX contracts and indexes. Review architecture, refresh handoff and run final checks.

## Interfaces and order

Task 2 supplies `port.Transaction` and `Transact(func(port.Transaction) error) error` for tasks 3 and 4.
Task 3 supplies `reminder.Service` to the worker and HTTP adapter.
Task 4 supplies provider implementations for process wiring; conversation owns asynchronous tasks and shutdown.
Task 5 changes Flutter controller constructor arguments only; wire contracts and rendered UI stay stable.
Task 6 verifies every introduced boundary and cross-package interface.

## Review focus

- Rollback after a reminder result or audit write fails must leave the occurrence active.
- Retry must return the recorded reminder result, including after a restart.
- Profile edits and recurring completion must update schedules atomically.
- Shutdown must finish or cancel accepted replies and worker tasks before closing SQLite.
- Dependency and DOX checks must detect imports, missing owners and broken child indexes.

## Commands and expected results

- `toolchain/architecture-check`: exit 0, no dependency violations.
- `toolchain/dox-check`: exit 0, complete ownership and links.
- `toolchain/run go go test -race ./...`: exit 0, all packages pass.
- `toolchain/run go go vet ./...`: exit 0.
- `toolchain/run go gofmt -l .`: no output.
- `toolchain/run flutter flutter test`: exit 0, tests and goldens pass.
- `toolchain/run flutter flutter analyze`: exit 0, no issues.
- `toolchain/run flutter dart format --set-exit-if-changed lib test`: zero changes.
- `toolchain/run flutter flutter build web --release --no-wasm-dry-run`: exit 0.
- Chat, reminder and calendar local checks: exit 0 against compiled web and temporary SQLite databases.
- `git diff --check`: exit 0.
