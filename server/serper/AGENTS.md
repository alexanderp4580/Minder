# Purpose

Serper HTTP adapter for the assistant's lookups.

# Ownership

The three searches and their reply decoding. Implements `assistant.Finder`.

# Local Contracts

One responsibility per file: `transport.go` (one authenticated POST, 10 s deadline, 64 KiB body, safe errors), `endpoint.go` (kind to path, credits and decoder: maps `/maps` 3, places `/places` 1, search `/search` 1), `decode.go` (trim a reply to at most three `assistant.Hit`s), `client.go` (compose them and log one line per lookup). Add a search by adding an endpoint row and, if its reply shape is new, a decoder.
Keep the key in memory. Errors and logs never carry the key, the query text or the provider body; the log line `serper lookup` records `kind`, `query_len`, `results`, `status`, `credits` and `duration_ms`, and `serper lookup failed` is a warning with the error.
Send `{q, gl: "nl", hl: "en"}` with header `X-API-KEY`. `Client.URL` is the base URL (default `https://google.serper.dev`). Fixtures use local HTTP servers; tests never call Serper.
Maps returns the full address with postal code and town and the phone; places returns one best match whose address is often short; search returns title, link and snippet.

# Work Guidance

Write failing behaviour tests before fixes. A test handler must read the request body before it waits on the request context.

# Verification

From the root: `toolchain/run go go test -race ./serper`.
Run `toolchain/architecture-check` for package boundaries.

# Child DOX Index

None.
