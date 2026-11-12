# Purpose

Executable process entrypoints.

# Ownership

Executable folders and their composition boundaries.

# Local Contracts

Wire application interfaces to infrastructure here. Keep business policy inside domain and application services.

# Work Guidance

Preserve cancellation and resource ownership.

# Verification

`toolchain/run go go test -race ./cmd/...`.

# Child DOX Index

- `minder/AGENTS.md`: Minder process and shutdown
