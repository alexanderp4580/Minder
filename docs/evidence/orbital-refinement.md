# Orbital visual refinement

Date: 2026-11-12. Branch: `m0-chat`. Starting commit: `5ad57f3`.

The owner asked to fix the visual gap identified in the comparison with the generated Orbital concept.
This refinement restores upper-right and lower-edge lighting, brighter arc highlights,
and separate orbit centres, sizes and tilts. Chat and Calendar retain distinct compositions.
Highlights fade through the middle to preserve text contrast. Cards remain opaque.

The shared painter retains three vector ellipses and a 180-second rotation.
Static lighting and foreground remain isolated from animation repaints.
No bitmap assets, blur filters, packages, application/controller changes or Go changes were added.

## Verification

| Check | Observed result |
|---|---|
| `toolchain/check` | Passed, exit 0: 39 Flutter tests, analysis/format, Go race/vet/format, hooks, architecture and DOX |
| Animation/lifecycle regression tests | Passed: movement without foreground rebuilds, distinct scenes, reduced motion, route hiding and app background/resume |
| Calendar and Chat goldens | Reviewed against the generated concept and updated |
| Release web build | Passed, exit 0 |
| `toolchain/calendar-local-check` | Passed: item placement and back navigation |
| `toolchain/reminder-local-check` | Passed: reminder card and Done action |
| `toolchain/local-check` with fixture | Passed: chat operations, offline retry and SQLite persistence at phone/desktop sizes |
| Brand guide and mockups | Rendered at 1280/390 px without document overflow or page errors; all mockups re-rendered at 2× with reduced motion |
| Shared guide/mockup CSS | Exact calendar/orbital style match |
| `toolchain/dox-check` and `git diff --check` | Passed |

[Compiled Calendar](local-calendar.png) and [compiled Chat](local-flow.png) show the implemented treatment.
The foundation and brand guide record the updated geometry and lighting rules.
Existing parent ownership and child indexes remain valid; UI/brand contracts describe the new treatment.
The older [0.4.0 evidence](orbital-backgrounds.md) records the superseded subtle treatment.

Phone frame timing and battery use remain unmeasured. Phone approval still gates merging.
This work does not push, merge or deploy the hosted app.

## APK 0.4.1

Release arm64 build passed, exit 0. APK signature verification passed and the certificate
matches 0.4.0. Application ID is `io.liveleds.minder`; split version code is `2010`.
Build uses the production API/auth defaults and the existing cached signing key.

Artifact: ignored `.local/artifacts/Minder-0.4.1-arm64.apk`, 18,597,574 bytes.
SHA-256: `d5c28cde2eacba61d5b96609cf42e3a1eaece8341fae6d801077c2e17ba9e17c`.

Builds retain the existing SDK Cupertino font and authentication plugin KGP migration warnings.
Neither failed verification or the release builds.

Telegram confirmed document delivery through the existing bot and configured chat.
