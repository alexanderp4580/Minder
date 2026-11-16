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

`agent_turn` rows are the durable agent queue. `EnqueueAgentTurn` stores an optional owner entry and a queued turn in one transaction; a repeated turn ID returns the stored turn. `ClaimAgentTurn` takes the oldest queued turn only while no other turn holds an unexpired lease, in one serialized transaction; an expired lease is claimed again. `ReleaseAgentTurn` requeues a claimed turn. `OwnerTurnPending` ignores source arrivals.

`source` rows hold `domain.SourceRef` records: one per forwarded email (created at ingestion; migration 006 backfills existing captures) and one per reviewed chat source. `FindSources(query)` matches label or sender of live sources and owner chat messages containing the query (LIKE wildcards literal), newest first. A blank query or more than 20 matches is an `Invalid` error, so no match is dropped silently. `OpenReviews` lists the newest ten live reviews waiting for the owner (status `needs-answer` or `ready-for-review`); `Entry` reads one chat entry by ID. `RedactCapture` also marks the source deleted, clears its label and sender and redacts a chat source's own entry.

`Board(now)` reads all item rows without the per-item source lookup (the board shows no sources) and groups them with `domain.BoardColumns`; the filtering and sorting rules stay in the domain.

# Work Guidance

Write failing behaviour tests before fixes. Preserve current wire and database contracts.

# Verification

From the root: `toolchain/run go go test -race ./store`.
Run `toolchain/architecture-check` for package boundaries.

# Child DOX Index

- `migrations/AGENTS.md`: numbered schema upgrades
