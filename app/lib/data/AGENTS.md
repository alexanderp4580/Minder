# Purpose

Client infrastructure adapters.

# Ownership

HTTP/SSE client, JSON mapping, OAuth and Android notification channel.

# Local Contracts

Implement domain interfaces. Keep routes, wire keys, native method names and serialization here.
ProviderScope owns shared API/auth clients. Channel callbacks detach when their controller disposes.
Do not embed server secrets in the client. Public Firebase identifiers are runtime data.

Capture routes use owner authentication. A review answer returns the owner's own entry. Attachment bytes download through the shared authenticated client; web Blob and Android FileProvider adapters save/open them without public URLs.


`UrlLinkLauncher` uses url_launcher in external-application mode. It rejects schemes outside HTTPS, tel and mailto, empty targets, hostless URLs and URLs with user information. Platform failures return false. `wire.dart` retains item notes, kind, reminder offsets and optional remind_at.


Item JSON preserves `phone`, `email`, `website` and `booking_reference`; `details` maps label/value pairs into typed values; absent keys keep older responses compatible.

Capture JSON retains the full candidate proposal as read-only detail fields; original evidence and subjects remain available in the source data.

`entryFromJson` reads the optional `options` list; an absent list is empty.

`ChatApi` also implements `BoardRepository`: `GET /v1/board`, `POST /v1/items` (title) and `POST /v1/items/{id}/status` (expected version and status). A 409 sets `RepositoryFailure.conflict`. `boardFromJson` tolerates a missing column.

# Work Guidance

Keep responsibilities focused. Preserve existing widget appearance during refactors.

# Verification

From root: `toolchain/run flutter flutter test`, `toolchain/run flutter flutter analyze`, `toolchain/architecture-check`.

# Child DOX Index

None.
