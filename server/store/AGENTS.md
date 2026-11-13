# Purpose

SQLite persistence adapter.

# Ownership

Database opening, SQL migrations, transactions, JSON rows, event cursors, schedule projections and delivery state.

# Local Contracts

Only this package imports SQLite or executes SQL. Implement application contracts without application imports.
Keep derived reminder schedules in the same transaction as item/profile writes. Domain owns projection rules.
Return domain not-found errors for missing rows. Keep storage failures distinct from conflicts.
No snooze choices, reminder messages, action orchestration or provider calls belong here.

ItemsBetween selects overlapping event/date ranges, including stays that start before the query window.

Capture rows persist jobs, leases, revisions and provenance. Ingestion receipts and source links are unique; ClaimCapture serializes lease acquisition.

Capture review push outbox rows are unique per source revision/device. Source deletion redacts history and retains minimal replay tombstones; file cleanup resumes after restart.

Item reads derive source IDs from live provenance links; deleted sources never appear as downloadable item attachments.

`PurgeItem(id, keepPending)` deletes an item with its reminder schedule, occurrences and push deliveries, earlier pending changes, audit rows, chat entries, events and source links in one transaction. It returns the removed entry IDs and the sources left without an item. It logs counts, never titles.

# Work Guidance

Write failing behaviour tests before fixes. Preserve current wire and database contracts.

# Verification

From the root: `toolchain/run go go test -race ./store`.
Run `toolchain/architecture-check` for package boundaries.

# Child DOX Index

- `migrations/AGENTS.md`: numbered schema upgrades
