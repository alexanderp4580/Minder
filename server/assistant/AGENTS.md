# Purpose

Model orchestration through model, proposal and persistence interfaces.

# Ownership

Model contract, system prompt, context selection, proposal batches and request cost accounting.

# Local Contracts

No HTTP client or provider SDK belongs here. Model implementations live in adapters.
Keep at most 200 open items in context. Never apply proposed changes. Reserve cost before calling the model.
Settle known provider costs; retain reservations for unknown costs. Put a reply cost on its first entry only.

Extraction asks as a thoughtful assistant would: any number of questions, no fixed field checklist, up to three quick replies each (`options`), notes as a short `- ` list, address in `location`, `reminder_offsets_minutes` from the allowed choices (invalid values are dropped). Email extraction uses capture-only context, source evidence and structured candidates/questions. Share usage accounting with normal chat; external text cannot authorize writes. Permit one validation correction per extraction batch; charge both calls and keep exact source-evidence validation and the ten-call source limit.

Chat tools are `propose_create`, `propose_update`, `propose_complete`, `propose_cancel` (status `cancelled`) and `propose_delete` (removes the item and its data), plus `propose_profile` and `list_items`. The prompt keeps done, cancelled and deleted apart. Deleted items and their cards leave the history and context. The agent sees the last 20 chat entries and at most 200 open items.

# Work Guidance

Write failing behaviour tests before fixes. Preserve current wire and database contracts.

# Verification

From the root: `toolchain/run go go test -race ./assistant`.
Run `toolchain/architecture-check` for package boundaries.

# Child DOX Index

None.
