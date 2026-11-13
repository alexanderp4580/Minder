# Calendar controls and separate booking details

App 0.9.0+25, branch `fix-connection-message`. The owner approved the controls/address guide and requested Phone, Email, Website and Booking reference outside Notes.

## Behavior

- Neutral square text actions, down/up disclosure chevrons, source-dialog right chevrons, white address links with Maps hint and external arrow.
- Shared theme and `ContentAction` define hover, press and keyboard focus. Canonical guide and shared mockup styles agree.
- Phone, email, website and booking reference are optional typed strings throughout extraction, proposals, item persistence and client models. They have their own rows in calendar and confirmation previews. Notes contain remaining instructions.
- Labelled older note metadata is displayed as separate fields without client writes. The server promotes it on edits; clear and Undo preserve the intended result. Conflicts, conditional call instructions and calendar UID/sequence Ref lines stay visible.
- References retain leading zeros and stay plain/selectable. Contact targets open only after a tap. Phone-format fixes cover adjacent labels, non-breaking spaces and optional international trunk markers.
- HTML conversion retains safe contact/link targets and separates table cells. No links are fetched. This server change deployed in `fc21e5e`.

## Observed verification

- Missing typed-field previews, overlong values, clear of legacy notes, UI contact rows and legacy preview separation failed before implementation, then passed.
- Real model: a synthetic HTML booking produced separate phone, email, website and leading-zero reference fields; confirmation retained them in temporary SQLite, with instructions in Notes. This is a single-case check, not a guarantee for arbitrary email.
- UI feature tests: 68 passed before final suite. Canonical guide rendered at 1280/390 px with no horizontal overflow; disclosure opened/closed. New calendar render inspected against the approved guide.
- `toolchain/check`: exit 0; 147 Flutter tests, analysis (no issues), format, Go race/vet/format, Worker, hooks, architecture, DOX and orbital exports passed.
- Release web built successfully. Chat, reminder, calendar and email browser scenarios exited 0; the email scenario checked separately saved phone/email/website/reference fields and remaining instructions in Notes.
- Release APK build: exit 0. Artifact `.local/artifacts/Minder-0.9.0-arm64.apk`, 19,355,430 bytes; SHA-256 `6ac0adbe83697c440158a9c79b3b50fd398b64231930677f2e254453dfbf9445`.
- `apksigner verify --print-certs`: exit 0; certificate SHA-256 `7bbc3aba5e417ffe87efa00c53c183c6490dc61905ce7231aaf99f87666d2572` matches 0.8.0. `aapt2` reports `io.liveleds.minder`, version 0.9.0, code 25.
- The initial Telegram attempt returned URLError. The owner authorized retry; Telegram confirmed message 2000 (19,355,430 bytes).
- Server implementation: `03c51fd`. Client/guide implementation: `04a3530`. No unresolved important findings in inline review. The owner authorized deployment; Coolify deployment `0t0sh9swt6xea8s2vsuwqlxq` finished on `fc21e5e` and the application reported running:healthy.

Phone verification remains pending. No real items were modified by checks.

## Email review follow-up: 0.9.1+26

The owner reported raw Dutch evidence in the email card and missing appointment duration suggestions. Expanded candidates now use shared labelled detail rows and neutral disclosure controls. Source evidence remains literal for validation and is not shown as the description. The review heading is English. Extraction translates titles, instructions, questions and options, preserving addresses, proper names and identifiers. Appointments without a stated end/duration ask how long they take; choices include 30 minutes and alternatives. Older duration questions without options receive a client fallback.

- Widget and server duration-fallback tests failed before the fixes, then passed. Inline choice submission checks the candidate ID, revision and answer. Regression checks also cover accessible booking references and preservation of supplied day/night choices.
- `toolchain/check`: exit 0; 150 Flutter tests, Go race/vet, analysis, format, Worker, hooks, architecture, DOX and orbital checks passed.
- Real model: synthetic Dutch booking instructions became English and typed contact/reference fields survived confirmation into temporary SQLite. A second Dutch booking without an end produced an English duration question with three suggestions, including 30 minutes. These are two cases, not a guarantee for arbitrary source material.
- Release web build: exit 0. Chat and reminder browser scenarios passed; calendar and email scenarios passed again after the accessibility fix. The email check covers parsed fields, the accessible reference, an inline duration reply, confirmation and saved SQLite fields. Browser assertions were corrected to inspect Flutter semantics rather than DOM text.
- The first APK build left the previous output; metadata verification caught version 0.9.0 before delivery. Rebuilt APK: exit 0, verified `io.liveleds.minder`, version 0.9.1, code 26. Certificate matches the prior APK.
- Final artifact: `.local/artifacts/Minder-0.9.1-arm64.apk`, 19,289,894 bytes; SHA-256 `8cfccc6f56cd44ef0e37a943e70817c8b4a6317d2b031965b3e7531943d79903`. Source: `d04c3a8`.
- Rendered synthetic check: [expanded email details and duration choices](email-review-details.png). Telegram confirmed one 0.9.1 upload: message 2001, 19,289,894 bytes. Coolify deployment `ory0emc19xfx2eroyyrh48k6` finished on exact commit `d04c3a89488fdc8669b3e4a80ef059e0c5f7f459`; application status was `running:healthy`. The branch is `fix-connection-message`, pinned to that commit. Pre-deploy readback recorded `fc21e5e` as the rollback commit. Main was not merged.

DOX pass: owning source/test/toolchain docs and the email mockup/spec were updated. Parent ownership and the approved brand guide remain unchanged because this patch adds no layer or visual rule.

Existing confirmed items are not rewritten. Original email and attachments remain downloadable. Phone verification remains pending.
