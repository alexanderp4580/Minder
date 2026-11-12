# Purpose

Minder process composition and lifecycle.

# Ownership

Settings, authentication, storage, services, provider wiring, listener and shutdown.

# Local Contracts

Keep Cloudflare headers and identity values out of main. Load authentication through access.
Cancel and wait for workers and accepted replies before closing SQLite. HTTP shutdown has a deadline.
Main logs process failures after run returns, so deferred cleanup executes.
Main builds the logger (`logging.go`), logs the effective settings with secrets reduced to set or unset flags, and routes HTTP server errors into the same log.
Shutdown logs each stage: signal, HTTP server, worker, conversation and database.

Opt-in email capture composes signed ingress, private files, document adapters and isolated model extraction. Shutdown waits for both email and reminder workers before SQLite closes.

# Work Guidance

Do not configure or deploy hosted services from this executable.

# Verification

`toolchain/run go go test -race ./cmd/minder` checks worker cancellation and shutdown ordering.

# Child DOX Index

None.
