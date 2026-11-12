# Purpose

Numbered SQLite schema upgrades.

# Ownership

`NNN_name.sql` files and their version order.

# Local Contracts

File N upgrades version N-1 to N. Apply each upgrade in one transaction.
Never edit released migrations; add a numbered file for a schema change. Refuse newer databases.

# Work Guidance

Keep data compatibility explicit in migration tests.

# Verification

`toolchain/run go go test -race ./store` covers upgrades and newer-schema refusal.

# Child DOX Index

None.
