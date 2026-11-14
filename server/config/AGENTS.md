# Purpose

Process setting validation.

# Ownership

Database/listener paths, model settings, cost values and Firebase runtime settings.

# Local Contracts

Read from an injected environment lookup. Reject unknown environments and custom production model endpoints.
Cost limits and conversion rates must be finite positive numbers. Errors identify settings, never credential values.
Private agent capture is off by default. `AGENT_TRACE_DIR` (absolute path) and `AGENT_TRACE_UNTIL` (RFC3339 timestamp) must be set together. Expired timestamps are valid so a restart does not re-enable collection.

Place lookup is opt-in through SERPER_API_KEY. SERPER_URL is the Serper base URL; it may differ only outside production (production allows `https://google.serper.dev`).

Email capture is opt-in through EMAIL_CAPTURE_ENABLED. Validate allowed senders, recipient, source directory and base64 HMAC key without logging their values.

# Work Guidance

Write failing behaviour tests before fixes. Preserve current wire and database contracts.

# Verification

From the root: `toolchain/run go go test -race ./config`.
Run `toolchain/architecture-check` for package boundaries.

# Child DOX Index

None.
