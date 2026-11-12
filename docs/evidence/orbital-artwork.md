# Authored Orbital artwork

Date: 2026-11-12. Branch: `m0-chat`. Starting commit: `cbeeda8`.

The owner approved reconstructing the Orbital artwork and animating highlights along fixed paths.
The middle panel of [the original concept board](orbital-concepts.png) is the reference.

## Artwork and rendering

[Calendar SVG](../../brand/orbital-calendar.svg) and [Chat SVG](../../brand/orbital-chat.svg)
are the canonical artwork. Curves use fixed Bézier paths in the 462×868 reference plane.
Visible ridge measurements and an overlay guided the trace; fitted ellipses were replaced where they missed the reference.
The Chat variant shifts and tightens the composition.

Atmospheric light, local edge gradients and soft edge glow stay in the static layer.
Short highlights travel along cached path metrics over 180 seconds. The artwork does not rotate.
Static artwork and foreground have separate repaint boundaries from the moving highlights.
Reduced motion hides the highlights. Hidden routes and backgrounded apps stop ticking.
No runtime image assets, SVG parser, package dependencies, API changes or Go changes were added.

[Reference / artwork / trace overlay](orbital-trace-overlay.png) compares the authoring plane at the same size.
Cyan lines in the third panel are diagnostic overlays. They are not app colours.
This is a visual reconstruction, not a pixel-identical reproduction of the generated bitmap.
The app retains its approved layout and fits the artwork to the viewport while preserving its aspect ratio.
[Compiled Calendar](local-calendar.png) and [compiled Chat](local-flow.png) show the actual Flutter web app.

[The exporter](../../toolchain/export-orbits.py) compiles SVG geometry, edge gradients and light placements
into Flutter data and embeds the same SVG artwork in the guide and mockups.
It validates the shared monochrome/glow renderer profile and rejects unsupported changes.
`toolchain/check` now rejects stale exports.

## Verification

| Check | Observed result |
|---|---|
| New composition test before implementation | Failed: old renderer had no independent static artwork layer |
| `toolchain/check` | Passed, exit 0: 40 Flutter tests, Go race/vet/format, Flutter analysis/format, hooks, architecture and DOX |
| Static composition regression | Static artwork pixels stay identical after 20 seconds while the full scene changes |
| Motion and lifecycle regressions | Passed: no foreground rebuilds from animation; distinct scenes, reduced motion and route/app visibility |
| Calendar and Chat goldens | Reviewed and updated |
| Export guard negative probes | Rejected stale Flutter output and an unsupported SVG colour; originals restored |
| `python3 toolchain/export-orbits.py --check` | Passed |
| Brand guide / mockups | Rendered at 1280/390 px without document overflow or page errors; all mockups re-rendered at 2× with reduced motion |
| Release web build | Passed, exit 0 |
| Calendar browser scenario | Passed: item placement and back navigation |
| Reminder browser scenario | Diagnostic run and original-scenario rerun passed; see initial timeout below |
| Full Chat browser scenario | Passed: Confirm, query, Discard, changes, Done, Undo, reload, offline retry, phone/desktop layout and SQLite persistence |
| Release arm64 APK build | Passed, exit 0 |
| APK signature and manifest | Verified; same certificate as prior APK, ID `io.liveleds.minder`, version `0.4.2`, split code `2011` |
| DOX, export consistency, shared CSS and `git diff --check` | Passed |

One initial reminder run timed out after the API reported an active reminder but the Done button
was not visible within 15 seconds. A diagnostic run and a rerun of the original scenario passed.
The cause was not reproduced or established; reminder code was not changed. The original failure log
and subsequent results remain under ignored `.local/orbital-trace/`.

The builds retain the existing SDK Cupertino font and authentication plugin KGP migration warnings.
Neither failed the release builds. Phone frame timing and battery use remain unmeasured.
Phone approval still gates merging. No push, merge or hosted deployment was performed.

## APK artifact

Ignored artifact: `.local/artifacts/Minder-0.4.2-arm64.apk`, 18,597,574 bytes.
SHA-256: `b48ac7590b711e572516837b611504566ac049fcb5f0fa0cd2e12ba312d251c2`.

Build uses production API/auth defaults and the existing cached signing key.

The first Telegram attempt reported a network error. A retry with the existing bot/chat configuration
and a longer upload timeout completed successfully; Telegram confirmed document delivery.
