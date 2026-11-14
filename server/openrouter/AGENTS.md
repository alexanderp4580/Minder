# Purpose

OpenRouter HTTP model adapter.

# Ownership

HTTP payloads, tool schema, token cap, response decoding and provider usage.

# Local Contracts

Implement the assistant model contract. Keep provider keys in memory.
Use request deadlines, bounded replies and safe provider errors. Reject truncated replies and missing or negative costs.
Provider fixtures use local HTTP servers. Real-model checks use temporary databases.

Tool schemas distinguish event ends, date-only ranges and deadlines; preserve travel zones and never invent missing clock times.

`propose_cancel` and `propose_delete` need `item_id` and `expected_version`, like `propose_complete`. `find_finished_items` takes an optional `query` and `list`. The tool set follows `Message.Purpose` on the first message (`toolsFor`): chat has the base tools; `PurposeOffer` adds `offer_lookup` (required `kind`, an enum built from `assistant.LookupKinds` whose description states each kind's cost and returned fields, and a required `query`); `PurposeEmail` has only `extract_capture`.


Item and capture tool schemas expose phone, email, website and booking_reference as separate strings and support clear flags. Notes hold remaining instructions.

# Work Guidance

Write failing behaviour tests before fixes. Preserve current wire and database contracts.

# Verification

From the root: `toolchain/run go go test -race ./openrouter`.
Run `toolchain/architecture-check` for package boundaries.

# Child DOX Index

None.
