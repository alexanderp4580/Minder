# Architecture cleanup

Approved by the owner on 2026-11-12, after review of the six-step scope.

## Outcome

Complete the Go dependency boundaries and Flutter cleanup. Install Agent0ai DOX across durable project boundaries.
Preserve HTTP contracts, existing SQLite schemas, reminder timing, cost accounting and UI rendering.
Work inline in `Minder-m0-chat` on `m0-chat`. Do not merge, deploy or build the stopped APK.

## Boundaries

- `domain`: values, validation, recurrence, reminder policy and identifiers. No infrastructure imports.
- `port`: persistence contracts and transaction interface, using domain values only.
- `command`: preview, apply, undo and profile changes through transaction interfaces.
- `reminder`: scheduling and reminder actions through transaction interfaces; delivery through a sender interface.
- `assistant`: prompt/context, proposals and costs through model and persistence interfaces.
- `conversation`: acceptance, concurrency, asynchronous reply lifecycle and safe error entries.
- `store`: SQLite adapter. Persist derived reminder projections in the same transaction as item/profile updates.
- `openrouter`, `push`, `access`: provider adapters. Dependencies point toward application contracts.
- `api`: HTTP adapter. Use application interfaces; do not import SQLite or concrete application implementations.
- `cmd/minder`: composition and lifecycle. Keep Cloudflare configuration in `access`.
- Flutter domain stays pure Dart. Controllers use domain repositories; data adapters own transport messages.

Transactions must roll back items, reminder state, previews, audit and events together on errors.
Pending changes remain durable and version-checked. Reminder actions remain idempotent.
Keep migration files unchanged. No new product dependencies or generic repository framework.

## DOX

Use the official `agent0ai/dox` framework, retaining Minder's user rules.
Each durable boundary documents purpose, ownership, contracts, guidance, verification and direct child indexes.
Keep tests and evidence with their owning boundary. Check indexes and links automatically.

## Acceptance

Enforce package/layer dependencies with an executable check. Check DOX coverage and indexes.
Run Go race tests, vet, formatting, Flutter tests, analysis, formatting, release web and all three browser scenarios.
Review the final tree against the boundaries above and resolve defects found by that review.
Keep phone-only validation and stopped APK work visible; automated checks cannot substitute for owner approval.
