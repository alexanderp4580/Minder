# Purpose

Client infrastructure adapters.

# Ownership

HTTP/SSE client, JSON mapping, OAuth and Android notification channel.

# Local Contracts

Implement domain interfaces. Keep routes, wire keys, native method names and serialization here.
ProviderScope owns shared API/auth clients. Channel callbacks detach when their controller disposes.
Do not embed server secrets in the client. Public Firebase identifiers are runtime data.

Capture routes use owner authentication. Attachment bytes download through the shared authenticated client; web Blob and Android FileProvider adapters save/open them without public URLs.


`UrlLinkLauncher` uses url_launcher in external-application mode. It rejects schemes outside HTTPS, tel and mailto, empty targets, hostless URLs and URLs with user information. Platform failures return false. `wire.dart` retains item notes, kind and reminder offsets.


Item JSON preserves `phone`, `email`, `website` and `booking_reference`; absent keys keep older responses compatible.

# Work Guidance

Keep responsibilities focused. Preserve existing widget appearance during refactors.

# Verification

From root: `toolchain/run flutter flutter test`, `toolchain/run flutter flutter analyze`, `toolchain/architecture-check`.

# Child DOX Index

None.
