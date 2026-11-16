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
- `model_fixture.py` supplies the test OpenRouter endpoint on loopback port 8181 and plays the one agent through the real tool loop: it reads a source arrival or pasted text with `read_source` and then calls `propose_review`, finds and reads a source to answer a question, and inspects a review before it updates one candidate for a reply. `GET /requests` lists the steps it played. It fails the first reading of a `failure-report-check` email with HTTP 500. Never run it in production.
- `board-local-check` and `board-check.cjs` run the Board on compiled web against a temporary server and SQLite database: add a task, Start, Undo, Start, Done, Reopen, then Calendar, Board and Chat navigation. They assert the board API counts after each step, no chat entries, six owner audit rows and a stored task without a due date or completion time.
- The calendar browser scenario checks selection, typed detail expansion/collapse and return to Chat.
- The chat browser scenario checks delayed offline status without a message or Retry block, restored composer text after failed sends, and recovery/resend through the header icon.
- `serper_fixture.py` supplies a fake Serper on a temporary port and lists the requests it received (`<path> <query>`) at `GET /calls`. `place-lookup-local-check` and `place-lookup-check.cjs` run the offer card against compiled web, the model fixture and that fake: the card names the search, the offer runs none, Look it up runs exactly that `/maps` search and the preview shows the postal code, Skip runs none. Never run the fixtures in production.
- `model_fixture.py` joins text blocks into plain text, because requests mark prompt-cache breakpoints with `cache_control`.
- Chat, reminder, calendar and email local checks each start an isolated model fixture on a temporary port. Run browser scenarios in sequence.
- `email-local-check` and `email-check.cjs` cover signed ingestion, the one-at-a-time review (the opening message, `Appointment i of N` and the `Done` summary, with a three-appointment email that includes a Teams meeting), compiled-web answers/downloads/review, the appointment flow (question message, expanded structured email details, targeted inline duration reply, reminder chip, Confirm and separate saved phone/email/website/booking-reference fields) and real SQLite provenance and reminder schedules. Duration answers reach the model fixture before a preview appears; the fixture returns the chosen duration. The email scenario verifies model-written openings and a meeting link in persisted `details`. It blocks the event stream for the first Confirm and checks that the original preview resolves through history. The model fixture has local pause/resume controls for review replies so the browser can verify Working while the model call remains pending. The email scenario also fails the first reading of a four-item email and checks that a visible failure report becomes the newest chat message and that Retry recovers it. It pastes a long (over 4,000 characters) three-appointment text into chat, expects the same review, answers a source question during the review without changing any candidate, answers one candidate by quick reply (only it changes), and deletes the pasted source.
- Real-model checks use temporary databases and provider cost values. `real-model-multi-check.py [runs]` checks two proposals.
- `real-model-agent-check --preparation` runs `real-model-preparation-check.cjs`: a long synthetic email with three appointments, stated lengths and a required questionnaire must produce four candidates with timing options one day, two days and a week before the first visit. It checks forwarded and pasted reviews before choosing a time, walks review order before answering the questionnaire, then sends a custom three-days-before answer. It requires the task scheduled on that day, the original deadline retained, the other proposals unchanged, a valid Confirm, and the actual SQLite notification schedule on the chosen work day. Test-side matching inspects expected choice labels/dates; no production timing parser is added.
- `real-model-multi-check.py` expects the two moved appointments at the next Monday 10:00 and Sunday 13:00 in Amsterdam, computed from today, compared as instants; the agent may use `start_at` and clear `due_at`.
- `real-model-board-check [runs]` runs the real model against a temporary server through chat only: an undated task stays undated, "I started on it" proposes `status` `in_progress`, and "put it back to open" proposes `open`, each confirmed and read back from `GET /v1/board`. `BOARD_CHECK_TRACE=1` keeps the private trace of a failed run in `/tmp/board-trace` and prints the model's version arguments. Costs about 0.006 USD a run.
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
