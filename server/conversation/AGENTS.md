# Purpose

Message acceptance and asynchronous reply lifecycle.

# Ownership

Text validation, busy state, accepted entry/reply IDs, reply timeout, panic recovery and shutdown.

# Local Contracts

Use repository and replier interfaces. Accept at most one active reply.
Preflight before saving a user entry. A cancelled HTTP request does not cancel an accepted reply.
Process shutdown cancels replies. `Close` stops acceptance and waits before storage closes.
Record safe error text for failed replies; never expose panic details.

# Work Guidance

Write failing behaviour tests before fixes. Preserve current wire and database contracts.

# Verification

From the root: `toolchain/run go go test -race ./conversation`.
Run `toolchain/architecture-check` for package boundaries.

# Child DOX Index

None.
