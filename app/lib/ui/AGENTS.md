# Purpose

Flutter screens and presentation.

# Ownership

ChatScreen, CalendarScreen, EntryView, time/cost labels and Tokens.theme.

# Local Contracts

Use controllers and typed domain values. Never build API routes or read raw transport maps.
Theme controls buttons, fields and standard visual density. Preserve 44 px controls and existing goldens.
Screens use Material icons and bundled fonts.

# Work Guidance

Keep responsibilities focused. Preserve existing widget appearance during refactors.

# Verification

From root: `toolchain/run flutter flutter test`, `toolchain/run flutter flutter analyze`, `toolchain/architecture-check`.

# Child DOX Index

None.
