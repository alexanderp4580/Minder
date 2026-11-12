# Calendar spacing evidence

Date: 2026-11-12. Branch: `m0-chat`. Starting commit: `aec916b`.

The owner approved the spacing preview before implementation.
The brand guide, foundation, calendar spec, mockup and Flutter page now use the approved layout.
The change stays in the presentation layer; repositories, controllers and Go code are unchanged.

## Observed checks

| Check | Result |
|---|---|
| Spacing regression before implementation | Failed: week pitch was 56 px instead of 64 px |
| `toolchain/check` | Passed, exit 0; 34 Flutter tests, Go race tests/vet, format, analysis, architecture, DOX and hook checks |
| Calendar golden | Reviewed and updated; chat golden unchanged |
| Narrow-screen widget checks | Passed at 320 and 360 px, 640 px high, with 1.4× text, long title, scrolling and Today action |
| Release web build | Passed, exit 0 |
| `toolchain/calendar-local-check` | Passed: confirmed item appears on its day; back returns to chat |
| Browser screenshots | Reviewed at 412×892 and 360×740, including the scrolled agenda |
| Brand guide | Rendered at 1280 and 390 px; no document overflow or page errors |
| HTML mockups | All re-rendered at device scale factor 2; calendar PNG refreshed |
| `git diff --check` | Passed |

The day targets remain at least 44 px wide at both narrow test widths.
Rows and selection frames grow for enlarged text. The whole page scrolls to preserve agenda space.
The normal-size selection frame is 44×44 px; the day target is 56 px high with 8 px between weeks.

The web build retains the SDK Cupertino font warning; the app uses Material icons.
The guide and HTML mockups retain their existing Carbon icons.

[Local calendar screenshot](local-calendar.png) shows the compiled Flutter web app with fixture data.
Additional screenshots and logs remain under ignored `.local/calendar-spacing-evidence/`.
These are local browser results. This slice did not deploy, push, merge or build an APK.
Owner phone approval still gates merging.
