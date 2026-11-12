# Purpose

Persistence interfaces owned by the application.

# Ownership

Transaction capabilities and `UnitOfWork`.

# Local Contracts

Depend on domain values only. `Transact` commits once on success and rolls back every write on error.
Transaction handles stay inside their callback. Read methods must return storage errors without inventing version conflicts.

# Work Guidance

Write failing behaviour tests before fixes. Preserve current wire and database contracts.

# Verification

From the root: `toolchain/run go go test -race ./port`.
Run `toolchain/architecture-check` for package boundaries.

# Child DOX Index

None.
