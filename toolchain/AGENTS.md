# Purpose

Pinned Podman runners, build helpers and verification scripts.

# Ownership

- `run`: Go and Flutter container runners and shared SDK caches.
- `check`: dependency/DOX checks, checker tests, hook tests, email Worker tests, Go checks and Flutter checks.
- `email-ingress-vectors.json`: synthetic cross-language HMAC fixture; no production addresses or secrets.
- `architecture-check`, `dox-check`, `tests/`: executable structure contracts and their regression tests.
- Local browser checks, model fixtures, real-model checks, callback probe and icon generation.

# Local Contracts

- Run commands from the repository root. `run go` starts in server/. `run flutter` starts in app/.
- Secrets come from process variables and never enter tracked files or browser processes.
- Named volumes cache dependencies, Android tools and the debug signing key. Preserve them for installable APK updates.
- Flutter uses Java 17, at most two Gradle workers and cached NDK 28.2.13676358.
- `model_fixture.py` supplies the test OpenRouter endpoint on loopback port 8181. Never run it in production.
- The calendar browser scenario checks selection, typed detail expansion/collapse and return to Chat.
- The chat browser scenario checks delayed offline status without a message or Retry block, restored composer text after failed sends, and recovery/resend through the header icon.
- `serper_fixture.py` supplies a fake Serper places endpoint on a temporary port and lists the queries it received at `GET /calls`. `place-lookup-local-check` and `place-lookup-check.cjs` run the offer card against compiled web, the model fixture and that endpoint: the offer runs no search, Look it up runs exactly one, Skip runs none. Never run the fixtures in production.
- Chat, reminder, calendar and email local checks each start an isolated model fixture on a temporary port. Run browser scenarios in sequence.
- `email-local-check` and `email-check.cjs` cover signed ingestion, compiled-web answers/downloads/review, the appointment flow (question message, expanded structured email details, targeted inline duration reply, reminder chip, Confirm and separate saved phone/email/website/booking-reference fields) and real SQLite provenance and reminder schedules.
- Real-model checks use temporary databases and provider cost values. `real-model-multi-check.py [runs]` checks two proposals.
- `android-callback-check.py` uses a disposable emulator and rejects physical device serials.
- `export-orbits.py` exports canonical brand SVG geometry/lighting to Flutter data and inline guide/mockup artwork. Its --check mode rejects stale exports in toolchain/check.
- `render-icons.py` exports assets from brand/app-icon.svg (layers `minder-orbit`, `minder-mark`) and requires rsvg-convert: Android adaptive bitmap layers, monochrome vector, legacy and web PNGs, favicon, grant logo. Do not hand-edit generated icons.
- `architecture-check` reads Go package imports and Dart directives. Inner code cannot import infrastructure implementations.
- `dox-check` validates direct child indexes, source ownership, section order and local Markdown links.

# Work Guidance

Use standard-library Python and existing pinned tools. Keep checks reproducible and errors specific.
The owner authorized orbital preview APK builds and Telegram delivery. Preserve the cached signing key. Do not change emulator or hosted services during local checks.

# Verification

`toolchain/check` runs the complete local static/unit suite. Release web and the browser scenarios (chat, reminder, calendar, place lookup) verify rendered integration.
Structure checker regressions run with `python3 -B -m unittest discover -s toolchain/tests -v`.
The callback check and phone notification checks remain separate from Flutter unit tests.

# Child DOX Index

None.
