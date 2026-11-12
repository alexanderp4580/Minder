# Purpose

Item and profile changes through transaction interfaces.

# Ownership

Preview batches, Confirm, Discard, Undo, conflict refresh, attribute application and preview fields.

# Local Contracts

Use `port.Transaction`; do not import storage implementations.
One invalid proposal rolls back its batch. Confirm checks versions and records audit/results atomically.
Pending changes do not expire. Undo refuses a changed current version. Preserve storage errors.

Event kind, end, zones and date-only attributes participate in previews, Confirm and Undo like existing item fields.

# Work Guidance

Write failing behaviour tests before fixes. Preserve current wire and database contracts.

# Verification

From the root: `toolchain/run go go test -race ./command`.
Run `toolchain/architecture-check` for package boundaries.

# Child DOX Index

None.
