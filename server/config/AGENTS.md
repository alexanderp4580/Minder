# Purpose

Process setting validation.

# Ownership

Database/listener paths, model settings, cost values and Firebase runtime settings.

# Local Contracts

Read from an injected environment lookup. Reject unknown environments and custom production model endpoints.
Cost limits and conversion rates must be finite positive numbers. Errors identify settings, never credential values.

# Work Guidance

Write failing behaviour tests before fixes. Preserve current wire and database contracts.

# Verification

From the root: `toolchain/run go go test -race ./config`.
Run `toolchain/architecture-check` for package boundaries.

# Child DOX Index

None.
