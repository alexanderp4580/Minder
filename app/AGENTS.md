# App

Flutter Android and web chat client. `domain` owns entries, the `ChatRepository` and `PushRepository` interfaces and `ReminderAction`.
`data` implements them and alone knows routes and wire names. `application` owns use cases and conversation state.
`ItemRepository` and the `LoadItems` use case feed the `CalendarController`; `CalendarScreen` is read-only and opens from the chat header.
`ui` calls controller methods and reads design tokens. It never builds a route.
Business rules stay on the server. Never embed server secrets in the client.
Run `../toolchain/run flutter flutter test` and `../toolchain/run flutter flutter analyze`.
Builds use bundled fonts. The app has no voice, Settings or preview Edit. `test/goldens/calendar.png` is the calendar reference.

Android browser callbacks use the authentication management activity from `flutter_web_auth_2` 5.1.
The exported callback uses an empty task affinity. The authentication manager uses `singleTask` to reuse the pending browser flow.
A Firefox callback must return to Minder and deliver its result.
OAuth client metadata includes the Minder site URL. Verify changes with `toolchain/android-callback-check.py` on a disposable emulator.

The launcher badge comes from `brand/app-icon.svg`. `toolchain/render-icons.py` exports adaptive, themed and legacy icons.
Web manifests and favicons use the same badge. OAuth client metadata includes both `client_uri` and `logo_uri`.
