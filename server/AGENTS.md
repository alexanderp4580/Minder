# Purpose

Go service: domain, application services and infrastructure adapters.

# Ownership

Package ownership and cross-package invariants. `go.mod` and `go.sum` pin server dependencies.

# Local Contracts

Dependencies point from adapters toward application contracts and domain values.
Only store writes SQL. Use real temporary SQLite files for transaction tests.
Store, provider and HTTP dependencies stay outside domain and application packages.
Logs use `log/slog`. Main sets the default logger from `LOG_LEVEL` (debug, info, warn or error; default info) and `LOG_FORMAT` (text or json; default text). Domain code never logs.
Info logs record every request, state change, model call, push attempt, reminder event and lifecycle event. Debug logs record transactions, saved entries, stream entries and worker ticks.
Each message names one event. Fields carry IDs, counts, sizes, `duration_ms` and error text. The request ID ties the lines of one request. `trace_id` and `reply_id` tie the lines of one assistant reply.
Never log secrets, tokens, JWTs, message text, item titles or notes. Log lengths and IDs. Identify a push token by `domain.Fingerprint`.
Tests capture logs with `slog.SetDefault` and a mutex-guarded buffer, and assert that private values stay out.

# Work Guidance

Use `toolchain/architecture-check` to enforce package dependencies.
Keep old migration files unchanged. Document runtime requirements in README.

# Verification

From the root: `toolchain/run go go test -race ./...`, `toolchain/run go go vet ./...`, and `toolchain/run go gofmt -l .`.
Formatting must print no paths.

# Child DOX Index

- `domain/AGENTS.md`: Pure Go values and business rules.
- `port/AGENTS.md`: Persistence interfaces owned by the application.
- `command/AGENTS.md`: Item and profile changes through transaction interfaces.
- `assistant/AGENTS.md`: Model orchestration through model, proposal and persistence interfaces.
- `conversation/AGENTS.md`: Message acceptance and asynchronous reply lifecycle.
- `reminder/AGENTS.md`: Reminder scheduling, actions and delivery orchestration.
- `capture/AGENTS.md`: Email capture orchestration.
- `mailauth/AGENTS.md`: Machine email authentication.
- `mailfiles/AGENTS.md`: Private persistent capture files.
- `store/AGENTS.md`: SQLite persistence adapter.
- `openrouter/AGENTS.md`: OpenRouter HTTP model adapter.
- `push/AGENTS.md`: Firebase Cloud Messaging adapter.
- `access/AGENTS.md`: Request authentication adapter.
- `config/AGENTS.md`: Process setting validation.
- `api/AGENTS.md`: HTTP transport adapter.
- `cmd/AGENTS.md`: process entrypoints
