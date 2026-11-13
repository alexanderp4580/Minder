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

The calendar items route accepts an optional validated IANA timezone for date-only overlap selection.

Email ingress is a separate machine-authenticated POST route with signed envelope/body verification; machine credentials never authorize owner routes. `POST /v1/changes/{id}/reminders` (body `{offsets}`) edits a pending change's reminders. Ingestion acknowledges only durable sources/jobs. Owner capture routes include answers, retry, delete and dismiss (`POST /v1/captures/{id}/dismiss`, body `{revision}`). Ingestion logs `email ingest accepted/rejected/failed` with reason codes and counts, never content.

`POST /v1/items/{id}/delete` (body `{expected_version}`, no other fields) returns a pending `delete_item` preview for the calendar. 404 for an unknown item, 409 for a stale version. The calendar confirms or discards it through the change routes.

# Work Guidance

Write failing behaviour tests before fixes. Preserve current wire and database contracts.

# Verification

From the root: `toolchain/run go go test -race ./api`.
Run `toolchain/architecture-check` for package boundaries.

# Child DOX Index

None.
