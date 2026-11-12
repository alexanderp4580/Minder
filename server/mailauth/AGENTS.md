# Purpose

Machine email authentication.

# Ownership

mailauth package.

# Local Contracts

Validate the signed outer envelope, body digest, timestamp and configured sender allowlist. Never grant owner route access or log configuration values.

# Work Guidance

Use synthetic fixtures; preserve original content and privacy.

# Verification

`toolchain/run go go test -race ./mailauth`.

# Child DOX Index

None.
