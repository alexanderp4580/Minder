# App icon redesign and faster highlights

Date: 2026-11-13. Branch: `app-icon-orbital` (from `notification-body-tap`, not merged, not pushed). Version 0.6.1+21, then 0.6.2+22.

The owner chose the Michroma M with a lit rim (variant C) over the Orbital background, then asked for 75 percent more intense background animation. A first change raised highlight alphas; the owner rejected it. Highlight brightness is unchanged. APK 0.6.1 made highlights 1.75 times faster (102.857 s loop). The owner then asked for another 50 percent: APK 0.6.2 loops in 68.571 s, 2.625 times the original 180 s speed. APK 0.6.1 and the first 0.6.2 draft moved linearly (`AnimationController.repeat()`, no curve). The owner then asked for ease in and out: the highlight painter now applies `Curves.easeInOut` to the loop value, so highlights slow near each loop point and peak at about 1.5 times the mean speed mid-loop.

## Icon

`brand/app-icon.svg` has two layers: `minder-orbit` and `minder-mark`. The M outline comes from `app/assets/fonts/Michroma-Regular.ttf`, 52 units wide. The icon has no Signal red.
`toolchain/render-icons.py` exports the Android adaptive background and foreground (bitmap), the monochrome vector, legacy and web PNGs, the favicon and the grant logo.
The adaptive background shrinks the orbit art to 75 percent so the lines remain inside circle and squircle masks. Mask previews were checked by eye at circle, squircle and 72 dp size; no physical launcher was checked.

## Checks

| Command or check | Observed result |
|---|---|
| New loop-speed test at the old 180 s duration | Failed as expected |
| Same test after the change to 102.857 s | Passed |
| `toolchain/check` | Passed, exit 0: 85 Flutter tests, analysis, format, Go, hooks, architecture, DOX and orbital export checks |
| `toolchain/run flutter flutter build apk --release --target-platform android-arm64` | Exit 0, 19,224,006 bytes |
| APK badging (aapt2) | `io.liveleds.minder`, version 0.6.1, version code 21 |
| APK signature (apksigner) | Verified; certificate SHA-256 `7bbc3aba…d2572` matches the previous APK |
| APK adaptive icon | Background, foreground and monochrome layers present |

APK 0.6.1 artifact: ignored `.local/artifacts/Minder-0.6.1-arm64.apk`. SHA-256: `5086969f0c2b4e51947fe05eadf8c414f1466027417dd04d0183c703eb2b3a39`.

## Delivery

Telegram delivery used the TokenCrypt secrets `telegram_astrabot_token` and `telegram_astrabot_chatid` and the existing bot and chat. Telegram confirmed one upload (message 1993, HTTP 200). A first attempt with a mistaken `--with` list returned 404 and delivered nothing.

APK 0.6.2+22 (ease in and out, 68.571 s loop): `.local/artifacts/Minder-0.6.2-arm64.apk`, 19,224,006 bytes, SHA-256 `ddbefd3499d56a89446612d6aa4daf4d583af0baf6e4a1de5e6953b8bab2584c`. `toolchain/check` exit 0 with 86 Flutter tests. aapt2 reports `io.liveleds.minder`, version 0.6.2, code 22; the signing certificate matches earlier APKs. Telegram confirmed one upload (message 1994).

## Limits

- Frame timing, battery use and the look on a physical launcher are unchecked. Phone approval is pending.
- Release web and the browser scenarios were not run; the change does not cross the client/server boundary.
