# Purpose

Flutter Android and web client.

# Ownership

Package/build configuration and the source, tests, native platform and asset folders.

# Local Contracts

Business validation runs on the server. The client displays typed results and errors.
The app has chat and a read-only calendar. Preserve current API contracts, spacing and notification actions.
The current app uses Material icons; Carbon remains a design-guide difference.

# Work Guidance

The runner starts in app/. Use format paths `lib test`. APK 0.4.0 with orbital backgrounds is authorized; preserve the cached signing key.

# Verification

From root: `toolchain/run flutter flutter test`, `toolchain/run flutter flutter analyze`,
`toolchain/run flutter dart format --set-exit-if-changed lib test`.
Release web and browser scenarios check client/server changes.

# Child DOX Index

- `lib/AGENTS.md`: source layers and composition
- `test/AGENTS.md`: controller, adapter and golden tests
- `android/AGENTS.md`: Android callbacks, notifications and launcher
- `web/AGENTS.md`: web bootstrap and generated icons
- `assets/AGENTS.md`: bundled fonts and licences
