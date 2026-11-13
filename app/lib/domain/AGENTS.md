# Purpose

Pure Dart values and interfaces.

# Ownership

Entry/item values, repository failures, chat/item/push repositories and native device contracts.

# Local Contracts

Import only domain files and Dart SDK libraries. No Flutter, HTTP, JSON factories or platform channels.
Use typed values for server and notification data.

Capture/source values and repository methods are typed; dates without times stay date-only and event overlap excludes checkout.


`Item` retains kind, notes and reminder offsets. `links.dart` finds visible phone, email and web spans and encodes Google Maps searches. `LinkLauncher` opens HTTPS, tel and mailto targets through a data adapter. Booking references and dates stay plain text, including numeric references with leading zeros in labelled booking/reference context. Phone matching accepts labelled numbers directly after a colon and non-breaking spaces. International “(0)” trunk markers drop from dial targets; visible source text stays unchanged.


`Item` carries phone, email, website and bookingReference as typed values. `BookingDetails` separates clearly labelled older contact/reference lines for display, keeps conflicts and calendar UID/sequence Ref lines in Notes, and never writes to the server.

# Work Guidance

Keep responsibilities focused. Preserve existing widget appearance during refactors.

# Verification

From root: `toolchain/run flutter flutter test`, `toolchain/run flutter flutter analyze`, `toolchain/architecture-check`.

# Child DOX Index

None.
