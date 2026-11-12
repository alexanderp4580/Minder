# Purpose

Firebase Cloud Messaging adapter.

# Ownership

Service account parsing, OAuth tokens, FCM payloads and invalid-token classification.

# Local Contracts

Keep service account data on the server. Validate the Google token endpoint before signing credentials.
Send high-priority data messages. Treat UNREGISTERED as an invalid device token; retry transient failures.

# Work Guidance

Write failing behaviour tests before fixes. Preserve current wire and database contracts.

# Verification

From the root: `toolchain/run go go test -race ./push`.
Run `toolchain/architecture-check` for package boundaries.

# Child DOX Index

None.
