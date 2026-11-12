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

# Work Guidance

Write failing behaviour tests before fixes. Preserve current wire and database contracts.

# Verification

From the root: `toolchain/run go go test -race ./store`.
Run `toolchain/architecture-check` for package boundaries.

# Child DOX Index

- `migrations/AGENTS.md`: numbered schema upgrades
