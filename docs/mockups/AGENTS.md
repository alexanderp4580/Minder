# docs/mockups

## Purpose

- HTML phone mockups (412×892 portrait) of every Minder screen, with a rendered PNG next to each

## Ownership

- `<screen>.html` and `<screen>.png` pairs, one per screen (index in `../screens.md`, "Mockups")
- `common.css`: shared tokens and components for every mockup

## Local Contracts

- Mockups follow `../screens.md`; no feature appears in a mockup that is not in the spec
- Mockups use the brand tokens, fonts and Carbon icons of `../../brand/index.html`: square corners, one cut corner on the primary button, Signal only for state and action
- Data is illustrative and marked so. Counts agree on every screen. Unknown details show as "?"
- A file with two states shows two phones side by side (`lists`, `profiles`)
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
