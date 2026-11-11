# App

Flutter Android and web chat client. `data` owns HTTP and DTO conversion.
`application` owns conversation state. `ui` reads controllers and design tokens.
Business rules stay on the server. Never embed server secrets in the client.
Run `../toolchain/run flutter flutter test` and `../toolchain/run flutter flutter analyze`.
Builds use bundled fonts. M0 has no voice, reminders, Settings or preview Edit.
