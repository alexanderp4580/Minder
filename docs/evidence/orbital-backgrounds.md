# Orbital backgrounds and APK evidence

Date: 2026-11-12. Branch: `m0-chat`. Starting commit: `3663740`.

The owner chose Orbital, approved distinct animated compositions for Chat and Calendar,
and requested APK delivery. Work stayed inline in the existing worktree.

## Implementation and checks

The presentation layer owns one shared painter component. It draws three thin ellipses
over a static halo. No bitmap assets, packages, API changes or Go changes were added.
The 180-second animation repaints arcs directly. Halo and foreground have separate repaint boundaries.

| Command or check | Observed result |
|---|---|
| Calendar background test before implementation | Failed: background absent |
| Lifecycle mutation check | Failed as expected when animation stopping was removed; original implementation restored |
| `toolchain/check` | Passed, exit 0; 39 Flutter tests, Go race/vet/format, Flutter analysis/format, hooks, architecture and DOX |
| Motion widget checks | Passed: rendered pixels change without foreground rebuilds; scenes differ |
| Accessibility and lifecycle checks | Passed: reduced motion stays static; hidden routes and backgrounded apps stop scheduling animation frames; visibility resumes motion |
| Goldens | Chat and Calendar reviewed and updated with reduced motion enabled |
| Release web build | Passed, exit 0 |
| `toolchain/calendar-local-check` | Passed: day placement and navigation back to Chat |
| `toolchain/reminder-local-check` | Passed: reminder card and Done action |
| `toolchain/local-check` with fixture | Passed: chat changes, Confirm, query, Discard, Done, Undo, reload, offline retry and SQLite persistence |
| Guide and HTML mockups | Rendered at 1280/390 px; no page errors or document overflow; all mockups re-rendered at 2× with reduced motion |
| Release arm64 APK build | Passed, exit 0 |
| APK validation | Signature verified, certificate matches the prior APK; application ID `io.liveleds.minder`, version `0.4.0`, split version code `2009` |
| `git diff --check` | Passed |

Artifact: ignored `.local/artifacts/Minder-0.4.0-arm64.apk`, 18,597,574 bytes.
SHA-256: `cd5a8a4245ec221b4a79a9b5b6c13ba242032f3a1d02aaae51c198ddef1a1c5b`.
Build used production API/auth defaults and the existing cached signing key.

The builds retain the SDK Cupertino font warning and the existing `flutter_web_auth_2`
Kotlin Gradle plugin migration warning. Neither failed the build.

Telegram delivery through the existing bot and configured chat completed successfully.
The delivery caption identifies this as the subtler implementation and leaves phone approval pending.

## Visual comparison and limits

The generated Orbital concept has brighter, more varied arcs and stronger light at both
the upper-right and bottom edge. The implementation deliberately reduced brightness,
but the comparison shows it loses some depth and visual impact. Stronger lighting and
more varied orbit centres are the recommended next refinement; they are not in this APK.

Current compiled captures: [Calendar](local-calendar.png), [Chat](local-flow.png).
The HTML mockups remain illustrative; Flutter goldens and compiled captures show the app.

Phone frame timing and battery use have not been measured. Local tests establish motion,
repaint isolation and lifecycle behaviour; they do not establish phone performance.
Phone visual/behaviour approval still gates merging to `main`.
No push, merge or hosted deployment was performed.
