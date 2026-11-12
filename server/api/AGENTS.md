# Purpose

HTTP transport adapter.

# Ownership

Routes, auth admission, body/range validation, response/error encoding, SSE and static assets.

# Local Contracts

Depend on query, command, message, reminder and authentication interfaces. Do not import adapters.
Only health and the OAuth logo are public. Same-origin checks protect browser writes.
Unknown API paths return 404. Stream durable cursors with 20-second pings.
Expose fixed messages for internal/model failures. Serve only named branding files.
`ServeHTTP` writes one `http request` line per request with status, size, duration, auth outcome and the real error cause. Handlers add inputs and output counts with `note`, never message text or secrets. Health checks log at debug.

# Work Guidance

Write failing behaviour tests before fixes. Preserve current wire and database contracts.

# Verification

From the root: `toolchain/run go go test -race ./api`.
Run `toolchain/architecture-check` for package boundaries.

# Child DOX Index

None.
