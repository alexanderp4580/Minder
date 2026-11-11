# Minder app

One Flutter chat screen for Android and web. The server owns item validation and changes.

Run Flutter commands through `../toolchain/run flutter` from this folder.
`flutter test` checks the controller, widgets, bundled-font golden and OAuth protocol boundaries.
`flutter analyze` checks Dart source.

Web uses the same origin and Cloudflare cookie. Android uses managed OAuth with PKCE and secure token storage.
The default Android API is `https://minder.liveleds.io`. A local build can set `API_BASE` and `LOCAL_AUTH` with Dart defines.

M0 includes text messages, previews, Confirm, Discard, Undo, Working and Retry. Voice and reminders come later.
