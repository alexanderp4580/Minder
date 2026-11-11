# Toolchain

Owns pinned Podman runners and local verification scripts. Run commands from the repository root.
`run go` mounts `server/`. `run flutter` mounts `app/`. Named volumes cache dependencies.
Secrets come from process environment and never enter tracked files.
Verify runners by building and testing both projects. Use `git diff --check` for scripts.

`model_fixture.py` provides the test-only OpenRouter boundary on loopback port 8181.
`local-check` uses a temporary database and random free port. `browser-check.cjs` drives compiled Flutter web.
The Flutter cache holds NDK 28.2.13676358. Android builds use Java 17 and at most two Gradle workers.
