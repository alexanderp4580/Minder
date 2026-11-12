# Purpose

Private persistent-volume capture files.

# Ownership

Atomic raw/decoded file writes, safe source identifiers, reads and deletion.

# Local Contracts

Never use sender filenames as paths. Enforce byte limits, write privately and sync before acknowledging persistence.
Application services own authorization, metadata and lifecycle. No SQL, model calls or HTTP belong here.

# Work Guidance

Use temporary directories and binary fixtures for tests.

# Verification

`toolchain/run go go test -race ./mailfiles`.

# Child DOX Index

None.
