# Server

Go HTTP service for M0. `domain` owns values. `store` alone writes SQL.
`command` owns item changes, previews, confirmation and undo. `assistant` proposes changes only.
`api` owns routes, authentication and event streams. `cmd/minder` wires the process.
All timestamps use UTC. The default display zone is Europe/Amsterdam.
Run `../toolchain/run go go test ./...` and `../toolchain/run go go vet ./...` from this folder.
Use test-first changes and real SQLite files for transaction tests.
