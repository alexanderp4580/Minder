# Purpose

Owner message acceptance and the serial agent queue worker.

# Ownership

Text validation, busy state, accepted entry/reply IDs, turn timeout, panic recovery and shutdown.

# Local Contracts

Use repository, agent and finisher interfaces. `Accept` trims text and accepts 1 to 400,000 UTF-8 bytes (the textual source budget). It rejects empty, oversized and invalid text before saving anything. It saves the user entry and a queued `owner` turn in one transaction and returns before the agent runs. The reply ID is the turn ID.
Accept rejects a second owner message while an owner turn is queued or active (`ErrBusy`). Source arrivals are queued by capture preparation and wait their turn; they are never dropped.
One worker claims turns oldest first and runs one at a time (`Agent.Run`, then `Finisher.CommitAgentTurn`). A failed run or commit goes to `Finisher.FailAgentTurn`; a panic fails the turn with `assistant panic`, never its details.
`Close` stops acceptance and cancels the worker. A turn cut short by shutdown returns to the queue and resumes after restart; it is not failed. Close waits before storage closes. A cancelled HTTP request does not cancel an accepted turn.

# Work Guidance

Write failing behaviour tests before fixes. Preserve current wire and database contracts.

# Verification

From the root: `toolchain/run go go test -race ./conversation`.
Run `toolchain/architecture-check` for package boundaries.

# Child DOX Index

None.
