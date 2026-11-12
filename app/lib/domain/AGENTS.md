# Purpose

Pure Dart values and interfaces.

# Ownership

Entry/item values, repository failures, chat/item/push repositories and native device contracts.

# Local Contracts

Import only domain files and Dart SDK libraries. No Flutter, HTTP, JSON factories or platform channels.
Use typed values for server and notification data.

Capture/source values and repository methods are typed; dates without times stay date-only and event overlap excludes checkout.

# Work Guidance

Keep responsibilities focused. Preserve existing widget appearance during refactors.

# Verification

From root: `toolchain/run flutter flutter test`, `toolchain/run flutter flutter analyze`, `toolchain/architecture-check`.

# Child DOX Index

None.
