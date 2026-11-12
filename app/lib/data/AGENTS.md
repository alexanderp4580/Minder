# Purpose

Client infrastructure adapters.

# Ownership

HTTP/SSE client, JSON mapping, OAuth and Android notification channel.

# Local Contracts

Implement domain interfaces. Keep routes, wire keys, native method names and serialization here.
ProviderScope owns shared API/auth clients. Channel callbacks detach when their controller disposes.
Do not embed server secrets in the client. Public Firebase identifiers are runtime data.

# Work Guidance

Keep responsibilities focused. Preserve existing widget appearance during refactors.

# Verification

From root: `toolchain/run flutter flutter test`, `toolchain/run flutter flutter analyze`, `toolchain/architecture-check`.

# Child DOX Index

None.
