# Calendar controls and separate booking details

App 0.9.0+25, branch `fix-connection-message`. The owner approved the controls/address guide and requested Phone, Email, Website and Booking reference outside Notes.

## Behavior

- Neutral square text actions, down/up disclosure chevrons, source-dialog right chevrons, white address links with Maps hint and external arrow.
- Shared theme and `ContentAction` define hover, press and keyboard focus. Canonical guide and shared mockup styles agree.
- Phone, email, website and booking reference are optional typed strings throughout extraction, proposals, item persistence and client models. They have their own rows in calendar and confirmation previews. Notes contain remaining instructions.
- Labelled older note metadata is displayed as separate fields without client writes. The server promotes it on edits; clear and Undo preserve the intended result. Conflicts, conditional call instructions and calendar UID/sequence Ref lines stay visible.
- References retain leading zeros and stay plain/selectable. Contact targets open only after a tap. Phone-format fixes cover adjacent labels, non-breaking spaces and optional international trunk markers.
- HTML conversion retains safe contact/link targets and separates table cells. No links are fetched. This server change requires deployment; current production remains unchanged.

## Observed verification

- Missing typed-field previews, overlong values, clear of legacy notes, UI contact rows and legacy preview separation failed before implementation, then passed.
- Real model: a synthetic HTML booking produced separate phone, email, website and leading-zero reference fields; confirmation retained them in temporary SQLite, with instructions in Notes. This is a single-case check, not a guarantee for arbitrary email.
- UI feature tests: 68 passed before final suite. Canonical guide rendered at 1280/390 px with no horizontal overflow; disclosure opened/closed. New calendar render inspected against the approved guide.
- `toolchain/check`: exit 0; 147 Flutter tests, analysis (no issues), format, Go race/vet/format, Worker, hooks, architecture, DOX and orbital exports passed.
- Release web built successfully. Chat, reminder, calendar and email browser scenarios exited 0; the email scenario checked separately saved phone/email/website/reference fields and remaining instructions in Notes.
- Release APK build: exit 0. Artifact `.local/artifacts/Minder-0.9.0-arm64.apk`, 19,355,430 bytes; SHA-256 `6ac0adbe83697c440158a9c79b3b50fd398b64231930677f2e254453dfbf9445`.
- `apksigner verify --print-certs`: exit 0; certificate SHA-256 `7bbc3aba5e417ffe87efa00c53c183c6490dc61905ce7231aaf99f87666d2572` matches 0.8.0. `aapt2` reports `io.liveleds.minder`, version 0.9.0, code 25.
- Telegram attempt returned URLError without a receipt. Connectivity to Telegram then succeeded. Delivery remains unconfirmed; owner confirmation was requested before a retry.
- Server implementation: `03c51fd`. Client/guide implementation: `04a3530`. No unresolved important findings in inline review. Production remains unchanged pending deployment authorization.

Phone verification and server deployment remain pending. No real items were modified by checks.
