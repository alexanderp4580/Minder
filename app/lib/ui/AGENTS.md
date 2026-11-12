# Purpose

Flutter screens and presentation.

# Ownership

ChatScreen, CalendarScreen, EntryView, OrbitalBackground, time/cost labels and Tokens.theme.

# Local Contracts

Use controllers and typed domain values. Never build API routes or read raw transport maps.
Theme controls buttons, fields and standard visual density. Preserve 44 px controls and existing goldens.
Screens use Material icons and bundled fonts.
CalendarTokens owns calendar dimensions. The calendar uses a centered, scrolling column with grouped month arrows and Today beside the agenda date.
Day targets stay at least 44 px wide; row height expands for enlarged text. Calendar goldens change only with an approved design.

OrbitalBackground owns decorative painting and animation lifecycle. Use distinct OrbitScene values for chat and calendar.
Keep halo and foreground isolated from animated arc repaints. Stop motion for reduced motion, hidden routes and backgrounded apps.

# Work Guidance

Keep responsibilities focused. Preserve existing widget appearance during refactors.

# Verification

From root: `toolchain/run flutter flutter test`, `toolchain/run flutter flutter analyze`, `toolchain/architecture-check`.

# Child DOX Index

None.
