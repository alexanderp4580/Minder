# App

Flutter Android and web chat client. `data` owns HTTP and DTO conversion.
`application` owns conversation state. `ui` reads controllers and design tokens.
Business rules stay on the server. Never embed server secrets in the client.
Run `../toolchain/run flutter flutter test` and `../toolchain/run flutter flutter analyze`.
Builds use bundled fonts. M0 has no voice, reminders, Settings or preview Edit.

Android browser callbacks use the authentication management activity from `flutter_web_auth_2` 5.1.
The exported callback uses an empty task affinity. The authentication manager uses `singleTask` to reuse the pending browser flow.
A Firefox callback must return to Minder and deliver its result.
OAuth client metadata includes the Minder site URL. Verify changes with `toolchain/android-callback-check.py` on a disposable emulator.
