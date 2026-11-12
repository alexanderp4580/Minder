# Purpose

Pure Go values and business rules.

# Ownership

Items, built-in profile defaults, proposals, chat values, recurrence, reminder policy, identifiers and operation errors.

# Local Contracts

Use the standard library only. Keep HTTP, SQL, SDKs and adapter imports out.
`Complete` advances recurrence; `SnoozeAt` uses the item zone. UTC storage times retain their item timezone.
`ProjectReminder` derives a schedule projection; persistence writes it atomically with item/profile changes.

# Work Guidance

Write failing behaviour tests before fixes. Preserve current wire and database contracts.

# Verification

From the root: `toolchain/run go go test -race ./domain`.
Run `toolchain/architecture-check` for package boundaries.

# Child DOX Index

None.
