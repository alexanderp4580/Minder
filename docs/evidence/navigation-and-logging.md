# Navigation, header and logging evidence

Date: 2026-11-12. Merged to `main` as `2033008`. Production runs that commit.

## Interface changes

- A two-item bottom bar (Chat, Calendar) replaces the header buttons.
- A 1 px rule that fades from left to right replaces the header border.
- The chat composer has 8 px above the field and 16 px below it.
- Orbital lines are 60 percent dimmer: the peak edge opacity fell from 0.6 to 0.24, and highlights scaled by the same factor.
- The bar items are 56 px high. The icon is 24 px, 8 px from the top. The label is 12 px, 8 px from the bottom.

The first bar version left each icon and label at the left edge of its item. A `Stack` loosened the constraints.
Renders with the test font hid the fault. A render of the compiled web app exposed it.
`bottom_nav_test.dart` now asserts the centre of every label and icon.
The owner reviewed earlier screenshots on Telegram. Review of APK 0.4.6 is pending.

## Checks

| Command or check | Observed result |
|---|---|
| `toolchain/check` on the merged tree | Passed, exit 0: 50 Flutter tests, Go race, vet and format, Flutter analysis and format, hooks, architecture, DOX and orbital export |
| Bar tests | Passed: items, selected state, 56 px targets, centring, 8 px balance, keyboard hiding, system inset below the bar |
| Header tests at 320, 412 and 1280 px | Passed: no button, no border, 1 px fading rule at the column width |
| Composer test | Passed: 68 px high, 16 px between field and bar |
| Orbital test | Passed: peak line opacity 0.24 |
| Release web build and real-font renders | Chat and Calendar inspected at 412 × 892 |
| Release arm64 APK 0.4.6+15 | Built; signing certificate matches earlier builds; SHA-256 `13732885…b6ea` |
| Telegram | One upload, HTTP 200, message 1967. Retries after 504 replies during the 0.4.5 upload sent the file five times |
| Android emulator | Not run. The emulator holds a different build of the app, version code 4002, so a downgrade install was not attempted |
| Phone | Layout and notifications not checked by the agent |

## Logging checks

| Command or check | Observed result |
|---|---|
| Per-package log tests | Passed in `api`, `access`, `openrouter`, `conversation`, `assistant`, `command`, `reminder`, `push`, `store`, `domain` and `cmd/minder` |
| Secret checks in those tests | Passed: message text, item titles, push tokens, API keys, JWTs and Firebase tokens never reach a log line |
| `go test -race ./...`, `go vet ./...`, `gofmt -l .` | Passed; `gofmt` printed no paths |
| Local debug run | Health, device registration, message with a refused model connection, bad request and shutdown logged. A search for the secret values returned no match |
| Architecture check | `push` and `access` import no internal package, so `push` keeps its own token fingerprint |

## Hosted result

Coolify deployment `wehc4yesilvupuyue3pzpp4v` finished for commit `2033008`. The app reports `running:healthy`.
The Coolify application now builds from branch `main`. It built from `m0-chat` before.
The first heartbeat in production logged `push_sender=true`, `items=5`, `devices=1` and `schedules_active=0`.

## Limits

- The deployment kept `revision=unknown` in `server configuration`, because the image build carries no Git data.
- Before this deployment the production log held one line, so the missing notifications today cannot be traced.
  The heartbeat shows one registered device and no waiting schedule. It cannot show whether a reminder fired earlier.
- Phone checks of the bar, the header rule and notifications remain with the owner.
