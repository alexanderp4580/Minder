# Purpose

Request authentication adapter.

# Ownership

Environment identity settings, assertion headers, JWT validation and signing-key cache.

# Local Contracts

Require explicit issuer, audience and owner email in production. Accept only signed, unexpired owner assertions.
The issuer is an HTTPS origin. Development auth requires an explicit development or test environment.
Keep provider headers and values inside this adapter, outside HTTP handlers and main.

# Work Guidance

Write failing behaviour tests before fixes. Preserve current wire and database contracts.

# Verification

From the root: `toolchain/run go go test -race ./access`.
Run `toolchain/architecture-check` for package boundaries.

# Child DOX Index

None.
