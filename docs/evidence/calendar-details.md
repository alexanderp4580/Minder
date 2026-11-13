# Calendar details, connection recovery and Confirmed footer

Worktree: `Minder-m0-chat`, branch `fix-connection-message`. App: 0.8.0+24. Implementation commit: `4f45e7e`.
The owner approved the details design, continued inline work and requested the Confirmed layout fix.

## Changes

- Calendar cards expand in place and show available start/end/deadline, duration, type, address, notes and reminders. Cross-zone items use Departs/Arrives; appointments use Starts/Ends.
- Address links open Google Maps. Visible phone, email and HTTPS links in notes open external apps. The launcher rejects other schemes and malformed targets. Failed launches show a snackbar.
- Notes keep the shared 600-character cap. Reference numbers and dates stay plain text.
- Confirmed has a separated footer with a check icon and semibold text.
- Connection recovery is included from `d0f0f68`.

## Observed checks

- Before completion, calendar tests failed because Details never rendered. Footer test failed because the Confirmed icon and emphasis were missing. HTTPS-only and empty-end regression tests failed before their fixes. Card-tap regression failed with the gesture disabled, then passed after restoring it.
- Feature tests: 62 passed (calendar, preview, parser and launcher); exit 0.
- Viewed the collapsed and expanded calendar goldens and the narrow Confirmed golden. The expanded calendar golden records all detail rows. The HTML mockup includes an expandable example.
- `toolchain/check`: exit 0 on 2026-11-13. Flutter 135 tests, analysis (no issues), format; Go race tests/vet/format; Worker, hooks, toolchain, architecture, DOX and orbital export checks passed.
- Release web and arm64 APK builds: exit 0. APK: `.local/artifacts/Minder-0.8.0-arm64.apk`, 19,289,794 bytes. SHA-256: `2958c92849d29d59acbaef8b3cedec00a995be5eb30f357ebc6e386f60229284`.
- `apksigner verify --print-certs`: exit 0. Certificate SHA-256 `7bbc3aba5e417ffe87efa00c53c183c6490dc61905ce7231aaf99f87666d2572` matches 0.7.0. `aapt2 dump badging` reports `io.liveleds.minder`, version 0.8.0, code 24.
- Chat browser/SQLite scenario: exit 0, including offline icon, no connection message or Retry, composer restoration, header recovery and resend. The original scenario still expected the removed message; the updated scenario verifies the current contract.
- Reminder browser scenario: exit 0 (Done, Snooze, Open and handling). Calendar browser scenario: exit 0 (selection, typed details expansion/collapse, return to Chat). The calendar accessibility group includes the card title and detail labels; the browser check reads that group.
- Telegram confirmed one upload: message 1998, 19,289,794 bytes. The sender used the existing TokenCrypt grants and made no retry.
- Phone opening of Maps, dialer and email and owner approval remain pending. No server deployment.
