# Toolchain

Owns pinned Podman runners and local verification scripts. Run commands from the repository root.
`run go` mounts `server/`. `run flutter` mounts `app/`. Named volumes cache dependencies.
Secrets come from process environment and never enter tracked files.
Verify runners by building and testing both projects. Use `git diff --check` for scripts.

`model_fixture.py` provides the test-only OpenRouter boundary on loopback port 8181.
`local-check` uses a temporary database and random free port. `browser-check.cjs` drives compiled Flutter web.
The Flutter cache holds NDK 28.2.13676358. Android builds use Java 17 and at most two Gradle workers.

The debug signing key lives in the named cache at `/cache/android/debug.keystore`.
Keep that cache for APK updates. Do not add the key to Git.

`real-model-check.py` starts a temporary local server with a real OpenRouter key from the environment.
`real-model-browser.cjs` checks preview, Confirm, query and Undo through compiled Flutter web.
The check requires real provider costs for each call and leaves no item in its temporary database.
The browser process receives no model key.

`android-callback-check.py` checks the browser-to-app callback on a disposable emulator.
It clears only the emulator installation of Minder and rejects physical device serials.
`app/test/native_auth_probe.dart` is its emulator-only entrypoint. The loopback fixture redirects from Firefox to the app.
The check asserts foreground ownership and callback delivery. It creates no account or authenticated session.

`real-model-multi-check.py [runs]` sends one message with two changes to the real model and checks for two previews with the right times. Run it after any change to the prompt, the tools or the context.

`render-icons.py` exports launcher, web, favicon and public-logo assets from `brand/app-icon.svg`. It requires `rsvg-convert`.
Do not edit generated icon files by hand. Rebuild exports after changing the SVG.

`reminder-local-check` and `calendar-local-check` start the model fixture and a server on a free port, then run `reminder-check.cjs` or `calendar-check.cjs` in compiled Flutter web.
`reminder-check.cjs` waits for a due reminder and presses Done. `calendar-check.cjs` opens the calendar page and checks that a confirmed item shows on its day.
`local-check` needs the fixture running on port 8181 before it starts.
