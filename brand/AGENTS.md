# Purpose

Brand guide website for Minder: colour, type, icons, layout, actions, cards, components, voice and platform notes.

# Ownership

This `brand/` folder owns `index.html`, `app-icon.svg`, its PNG preview and the canonical `orbital-calendar.svg` / `orbital-chat.svg` artwork. The page uses inline CSS and a small script. The script marks the current section in the navigation, places the layout pins and labels table cells on narrow screens.

# Local Contracts

- The page follows its own guide: Void/Signal palette, Michroma titles, Chakra Petch text, edge column per section, focus frame, square progress row.
- Colour values, sizes and contrast ratios on the page are the brand source of truth; notification and client designs copy them from here.
- Signal red has five uses: focus frame, overdue or urgent state, primary action, selected or active state, live recording. Everything else is neutral. No text is below 12 px. Every control has a 44 × 44 px hit area. `--edge` is for outlines only, never text.
- The Components section mirrors the component rules of `docs/mockups/common.css`; change both in the same edit.
- Chat and Calendar share the Calendar header treatment, one title row and a rule below it that fades left to right. Both use the same spacing. Headers have no buttons. Keep presence/status copy out of headers. A shared two-item bottom bar (Chat, Calendar) is 56 px high.
- The calendar example defines 8/16/24/32 px spacing, grouped month arrows, an inset selection frame and a scrolling agenda. Keep it aligned with Flutter and the shared mockup styles.
- Orbital SVGs own fixed paths, lighting and edge gradients in a 462×868 reference plane. Small highlights travel along paths over about 69 seconds, easing in and out. Honour reduced motion.
- The concept reference is docs/evidence/orbital-concepts.png. SVGs are authoring inputs; the app compiles exported paths and the guide embeds exported SVGs. No runtime asset route is needed.
- Run `python3 toolchain/export-orbits.py` after artwork edits. It exports Flutter geometry/lighting and inline SVGs in the guide and mockups. Do not hand-edit exports.
- Items not yet verified carry "to check" or "Design pending".
- Fonts load from Google Fonts. Icons load from jsDelivr (@carbon/icons, pinned version). The primary logo is the word Minder. App launchers and OAuth consent use the app icon in `app-icon.svg`.

# Work Guidance

Describe only what exists. Write page copy at about 80% of ASD-STE100 (see root AGENTS.md) and use stop-slop. Re-read the whole page after editing and check desktop (1280 px) and mobile (390 px) renders.

# Hosting

The Go service serves `index.html`, `app-icon.svg` and `app-icon.png` at `/branding/`, behind the same sign-in as the app. Add a file to the guide in `server/api/branding.go` and the Dockerfile, or it stays unreachable. The public grant logo at `/brand/minder-icon.png` is separate.

# Verification

Serve locally with `python3 -I -m http.server 8899 --directory brand` and open `http://127.0.0.1:8899/`.

# Child DOX Index

No child instruction files.

`app-icon.svg` is the canonical app icon. `toolchain/render-icons.py` exports all Android, web and public grant-logo assets.
It has two layers: `minder-orbit` (Void with the Orbital halo and three thin ellipses) and `minder-mark` (the Michroma M outline with a soft rim glow, a Text-colour body that dims away from the top right, and a thin bright rim). The M path is converted from `app/assets/fonts/Michroma-Regular.ttf` (OFL). It is 52 units wide, inside the Android adaptive safe zone. The icon has no Signal red.
