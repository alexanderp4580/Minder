# Purpose

Serper places HTTP adapter.

# Ownership

The places request, response decoding and its log line. Implements `assistant.PlaceFinder`.

# Local Contracts

Keep the key in memory. Errors and logs never carry the key, the query text or the provider body; the log line `place lookup` records `query_len`, `places`, `status` and `duration_ms`.
Send `{q, gl: "nl", hl: "en"}` with header `X-API-KEY`. Use a 10 s deadline, read at most 64 KiB and keep at most three places. Fixtures use local HTTP servers; tests never call Serper.

# Work Guidance

Write failing behaviour tests before fixes. A test handler must read the request body before it waits on the request context.

# Verification

From the root: `toolchain/run go go test -race ./serper`.
Run `toolchain/architecture-check` for package boundaries.

# Child DOX Index

None.
