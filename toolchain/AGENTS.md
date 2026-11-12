# Purpose

Pinned Podman runners, build helpers and verification scripts.

# Ownership

- `run`: Go and Flutter container runners and shared SDK caches.
- `check`: dependency/DOX checks, checker tests, hook tests, Go checks and Flutter checks.
- `architecture-check`, `dox-check`, `tests/`: executable structure contracts and their regression tests.
- Local browser checks, model fixtures, real-model checks, callback probe and icon generation.

# Local Contracts

- Run commands from the repository root. `run go` starts in server/. `run flutter` starts in app/.
- Secrets come from process variables and never enter tracked files or browser processes.
- Named volumes cache dependencies, Android tools and the debug signing key. Preserve them for installable APK updates.
- Flutter uses Java 17, at most two Gradle workers and cached NDK 28.2.13676358.
- `model_fixture.py` supplies the test OpenRouter endpoint on loopback port 8181. Never run it in production.
- `local-check` needs the fixture running. Reminder/calendar scripts start their own fixture; run them in sequence.
- Real-model checks use temporary databases and provider cost values. `real-model-multi-check.py [runs]` checks two proposals.
- `android-callback-check.py` uses a disposable emulator and rejects physical device serials.
- `render-icons.py` exports assets from brand/app-icon.svg and requires rsvg-convert. Do not hand-edit generated icons.
- `architecture-check` reads Go package imports and Dart directives. Inner code cannot import infrastructure implementations.
- `dox-check` validates direct child indexes, source ownership, section order and local Markdown links.

# Work Guidance

Use standard-library Python and existing pinned tools. Keep checks reproducible and errors specific.
Do not build the stopped APK unless the owner asks. Do not change emulator or hosted services during local checks.

# Verification

`toolchain/check` runs the complete local static/unit suite. Release web and the three browser scenarios verify rendered integration.
Structure checker regressions run with `python3 -B -m unittest discover -s toolchain/tests -v`.
The callback check and phone notification checks remain separate from Flutter unit tests.

# Child DOX Index

None.
