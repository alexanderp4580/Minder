# Purpose

Brand guide website for Minder: colour, type, layout, actions, cards, voice and platform notes.

# Ownership

This `brand/` folder owns `index.html`, a single static page with inline CSS and a small script. The script marks the current section in the navigation, places the layout pins and labels table cells on narrow screens.

# Local Contracts

- The page follows its own guide: Void/Signal palette, Michroma titles, Chakra Petch text, edge column per section, focus frame, square progress row.
- Colour values, sizes and contrast ratios on the page are the brand source of truth; notification and client designs copy them from here.
- Items not yet verified carry "to check" or "Design pending".
- Fonts load from Google Fonts. Icons load from jsDelivr (@carbon/icons, pinned version). The logo is the word Minder, with no icon.

# Work Guidance

Describe only what exists. Write page copy at about 80% of ASD-STE100 (see root AGENTS.md) and use stop-slop. Re-read the whole page after editing and check desktop (1280 px) and mobile (390 px) renders.

# Verification

Serve locally with `python3 -I -m http.server 8899 --directory brand` and open `http://127.0.0.1:8899/`.

# Child DOX Index

No child instruction files.
