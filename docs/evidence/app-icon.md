# App icon and orbital highlights

Date: 2026-11-13. Version 0.6.2+22. Merged into local `main`; not pushed.

## Current state

The launcher icon is the Michroma M with a lit rim over the Orbital background. It has no Signal red.
`brand/app-icon.svg` has two layers, `minder-orbit` and `minder-mark`. The M outline comes from `app/assets/fonts/Michroma-Regular.ttf` and is 52 units wide.
`toolchain/render-icons.py` exports the Android adaptive background and foreground as bitmaps, the monochrome vector, the legacy and web PNGs, the favicon and the grant logo.
The adaptive background shrinks the orbit art to 75 percent so the lines stay inside circle and squircle masks.

The orbital highlights ease in and out (`Curves.easeInOut`) over a 68.571 s loop. That is 2.625 times the speed of the original 180 s linear loop. Peak speed mid-loop is about 1.5 times the mean. Highlight brightness did not change.

## Checks

| Check | Result |
|---|---|
| Loop-time test at the old 180 s duration | Failed, as expected |
| Ease test at linear motion | Failed, as expected |
| Both tests after the change | Passed |
| `toolchain/check` | Exit 0, 86 Flutter tests, with analysis, format, Go, hooks, architecture, DOX and orbital export checks |
| `toolchain/run flutter flutter build apk --release --target-platform android-arm64` | Exit 0, 19,224,006 bytes |
| aapt2 badging | `io.liveleds.minder`, version 0.6.2, code 22 |
| apksigner | Verified. The certificate SHA-256 `7bbc3aba…d2572` matches earlier APKs |

Artifact: `.local/artifacts/Minder-0.6.2-arm64.apk` (ignored by Git). SHA-256 `ddbefd3499d56a89446612d6aa4daf4d583af0baf6e4a1de5e6953b8bab2584c`.
Telegram confirmed one upload of 0.6.2 (message 1994). Delivery used the TokenCrypt secrets `telegram_astrabot_token` and `telegram_astrabot_chatid`. APK 0.6.1 reached Telegram earlier (message 1993).

## Limits

- Nobody has checked the icon on a physical launcher, or frame timing and battery use. Phone approval is pending.
- Release web and the browser scenarios did not run for this change. It does not cross the client/server boundary.
