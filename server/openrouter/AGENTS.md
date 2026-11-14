# Purpose

OpenRouter HTTP model adapter.

# Ownership

HTTP payloads, tool schema, token cap, response decoding and provider usage.

# Local Contracts

Implement the assistant model contract. Keep provider keys in memory.
`trace.go` owns optional private diagnostics. `NewTrace` secures the directory to 0700 and rejects a symlink directory; each event is a complete JSON file created at 0600. Requests include the wire body only, not HTTP headers or client configuration. Responses retain the body within the existing 2 MiB read limit and mark when that limit is reached; raw unreadable and rejected replies are captured too. Turn IDs and generated call IDs correlate events without entering filenames. Check the capture deadline before each event, including responses and agent diagnostics. Capture write failures emit only safe metadata and do not fail a model call. Expiry stops collection; existing files remain for inspection and explicit cleanup.
Use request deadlines, bounded replies and safe provider errors. Send `max_tokens` 100,000 (the model allows 128,000), a ceiling no normal reply reaches; reject truncated replies (`ReasonCutOff`) and missing or negative costs.
Provider fixtures use local HTTP servers. Real-model checks use temporary databases.

Tool schemas distinguish event ends, date-only ranges and deadlines; preserve travel zones and never invent missing clock times.

`propose_cancel` and `propose_delete` need `item_id` and `expected_version`, like `propose_complete`. `find_finished_items` takes an optional `query` and `list`. One tool family serves every request (`toolsFor`): the chat tools, the source tools (`find_sources`, `read_source`, `inspect_review`, `propose_review`) and, when `Message.Purpose` on the first message is `PurposeOffer`, `offer_lookup` (required `kind`, an enum built from `assistant.LookupKinds` whose description states each kind's cost and returned fields, and a required `query`). Messages carry tool calls and results with their IDs (`tool_calls` on assistant messages, `role: tool` with `tool_call_id`). `propose_review` takes `source_id`, `expected_revision`, `summary`, `notification` and `candidates` (optional `id` and `revision` to change one, `proposal`, `question`, `options`, `evidence`).


Item and review tool schemas expose phone, email, website and booking_reference as separate strings and support clear flags. Item tools also expose `details` (label/value pairs) and `clear_details`. Notes hold remaining instructions.

# Work Guidance

Write failing behaviour tests before fixes. Preserve current wire and database contracts.

# Verification

From the root: `toolchain/run go go test -race ./openrouter`.
Run `toolchain/architecture-check` for package boundaries.

# Child DOX Index

None.
