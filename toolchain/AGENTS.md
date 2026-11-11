# Toolchain

Owns pinned Podman runners and local verification scripts. Run commands from the repository root.
`run go` mounts `server/`. `run flutter` mounts `app/`. Named volumes cache dependencies.
Secrets come from process environment and never enter tracked files.
Verify runners by building and testing both projects. Use `git diff --check` for scripts.
