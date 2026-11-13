# docs/mockups

## Purpose

- HTML phone mockups (412×892 portrait) of the chat and calendar layouts, with rendered PNGs

## Ownership

- `chat.html` and `chat.png`: the chat design draft (spec in `../screens.md`); Flutter goldens record current rendering
- `email-capture.html` and `email-capture.png`: English source summary, expanded structured fields, attachments and targeted duration choices in existing chat; governed by the email capture spec.
- `place-lookup.html` and `place-lookup.png`: draft of the Serper place-lookup offer, result preview and no-match states; governed by `../superpowers/specs/2026-11-13-place-lookup-design.md`
- `calendar.html` and `calendar.png`: the approved month grid and agenda layout. The illustrative marker sits above the app frame. The agenda card shows Delete and a neutral chevron disclosure, source-zone time rows, an address block with Maps destination and white note links. Shared `.content-*` rules match the canonical guide.
- `today`, `today-states`, `board`, `lists`, `item`, `profiles`, `assistant`, `settings`: design drafts that do not define current app behaviour. Use `../screens.md` and the Flutter calendar golden for the implemented calendar
- `common.css`: shared tokens and components for every mockup, including calendar layout and fixed orbital surfaces mirrored in the brand guide

## Local Contracts

- The mockup follows `../screens.md` and `../foundation.md`; no feature appears in it that is not in the spec
- Mockups use the brand tokens, fonts and Carbon icons of `../../brand/index.html`: square corners, one cut corner on the primary button. Signal red appears only for the focus frame, overdue or urgent state, the primary action, the selected or active state and live recording. No text is below 12 px and every control has a 44 × 44 px hit area
- Data is illustrative. Each mockup shows one `.demo` tag in its status bar, or above the Chat/Calendar frame. That tag is the only place the word "illustrative" appears in a mockup. Counts agree on every screen. Unknown details show as "?"
- A file with two states shows two phones side by side (`lists`, `profiles`)
- `toolchain/export-orbits.py` owns inline orbital SVGs from brand/orbital-*.svg. Regenerate them after source artwork edits.
- Chat and Calendar share the screen-header styles and fading rule; neither shows a presence/status line or a header button. Both end with the shared `.nav` bar (Chat, Calendar).
- Chat and Calendar have distinct fixed orbital compositions with travelling highlights. Capture stills with reduced motion; Flutter owns app lifecycle pauses.
- Every PNG is re-rendered after its HTML or `common.css` changes

## Work Guidance

- Fonts and icons load from Google Fonts and jsDelivr; rendering needs a network connection
- Render with Playwright (Chromium, `--no-sandbox`), viewport 412×892, device scale factor 2, 2.5 s wait for fonts and icons. For several phones, widen the viewport to the width of `.page`
- Render command (script saved outside the project, for example `/tmp/render.js`; arguments: this folder, then screen names without extension):
  `PATH=/home/deck/.nvm/versions/node/v22.23.2/bin:$PATH node /tmp/render.js "$PWD" today board`
- The script loads the `playwright` module from `/home/deck/SketchBook/Projects/AstraMoode/AstraCamillaGui/node_modules/playwright` and launches the Chromium at `/home/deck/.cache/ms-playwright/chromium-1243/chrome-linux64/chrome`. It opens each HTML, waits 2.5 s, sets the viewport width to the width of `.page`, and writes the PNG next to the HTML

## Verification

- Open every PNG: no overflow, no clipped text, no rounded card corners, fonts and icons loaded, no "undefined"
- Check counts and names against `../screens.md`

## Child DOX Index

- None
