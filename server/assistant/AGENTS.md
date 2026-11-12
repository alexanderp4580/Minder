# Purpose

Model orchestration through model, proposal and persistence interfaces.

# Ownership

Model contract, system prompt, context selection, proposal batches and request cost accounting.

# Local Contracts

No HTTP client or provider SDK belongs here. Model implementations live in adapters.
Keep at most 200 open items in context. Never apply proposed changes. Reserve cost before calling the model.
Settle known provider costs; retain reservations for unknown costs. Put a reply cost on its first entry only.

Email extraction uses capture-only context, source evidence and structured candidates/questions. Share usage accounting with normal chat; external text cannot authorize writes.

# Work Guidance

Write failing behaviour tests before fixes. Preserve current wire and database contracts.

# Verification

From the root: `toolchain/run go go test -race ./assistant`.
Run `toolchain/architecture-check` for package boundaries.

# Child DOX Index

None.
