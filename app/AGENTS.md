# App

Flutter Android and web client: the chat screen and a read-only calendar page.

## Layers

- `domain`: entries, `Change`, `ReminderInfo`, items, the repository interfaces (`ChatRepository`, `PushRepository`, `ItemRepository`) and `ReminderAction`.
- `data`: implements the interfaces. It alone knows routes and wire names.
- `application`: use cases and controllers (`ChatController`, `CalendarController`, `NotificationController`).
- `ui`: screens and widgets. They call controller methods, read typed models and design tokens, and never build a route or read a raw JSON map.
- `ui/entry_view.dart` draws one chat entry. `ui/format.dart` owns time, cost and label text.

## Look

`Tokens.theme` owns the look of buttons, icon buttons and fields. Screens use plain Material widgets and set no per-widget styles. The theme pins `VisualDensity.standard`, because desktop browsers default to compact density and shrink fields below 44 px.
`test/ui_spec_test.dart` checks the spacing numbers of `docs/foundation.md`. Goldens: `test/goldens/m0-chat.png` for chat, `test/goldens/calendar.png` for the calendar.

## Checks

Run these from the repository root; each must pass before a change is done:
- `toolchain/run flutter flutter test`
- `toolchain/run flutter flutter analyze`
- `toolchain/run flutter dart format --set-exit-if-changed lib test`

`toolchain/run flutter` starts in `app/`, so format paths are `lib test`, not `app/lib`.
Business rules stay on the server. Never embed server secrets in the client. Builds use bundled fonts. The app has no voice, Settings or preview Edit.

Android browser callbacks use the authentication management activity from `flutter_web_auth_2` 5.1.
The exported callback uses an empty task affinity. The authentication manager uses `singleTask` to reuse the pending browser flow.
A Firefox callback must return to Minder and deliver its result.
OAuth client metadata includes the Minder site URL. Verify changes with `toolchain/android-callback-check.py` on a disposable emulator.

The launcher badge comes from `brand/app-icon.svg`. `toolchain/render-icons.py` exports adaptive, themed and legacy icons.
Web manifests and favicons use the same badge. OAuth client metadata includes both `client_uri` and `logo_uri`.
