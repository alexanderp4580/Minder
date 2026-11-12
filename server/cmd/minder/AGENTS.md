# Purpose

Minder process composition and lifecycle.

# Ownership

Settings, authentication, storage, services, provider wiring, listener and shutdown.

# Local Contracts

Keep Cloudflare headers and identity values out of main. Load authentication through access.
Cancel and wait for workers and accepted replies before closing SQLite. HTTP shutdown has a deadline.
Main logs process failures after run returns, so deferred cleanup executes.

# Work Guidance

Do not configure or deploy hosted services from this executable.

# Verification

`toolchain/run go go test -race ./cmd/minder` checks worker cancellation and shutdown ordering.

# Child DOX Index

None.
