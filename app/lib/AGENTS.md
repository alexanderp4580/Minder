# Purpose

Flutter client source.

# Ownership

`main.dart` wires providers, navigation and app lifecycle. Child layers own feature code.

# Local Contracts

Keep concrete data adapters in composition or the data layer.
ProviderScope owns shared API/auth clients and closes them on disposal.
Keep pure domain models independent from Flutter.

# Work Guidance

Use controller tests and adapter boundary tests for lifecycle changes.

# Verification

`toolchain/architecture-check` and Flutter tests/analyze.

# Child DOX Index

- `domain/AGENTS.md`: Pure Dart values and interfaces.
- `application/AGENTS.md`: State and lifecycle orchestration.
- `data/AGENTS.md`: Client infrastructure adapters.
- `ui/AGENTS.md`: Flutter screens and presentation.
