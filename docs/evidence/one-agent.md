# One agent evidence

Source: branch `guided-email-review`, commits after `dadf13d`. Design and plan: [design](../superpowers/specs/2026-11-14-one-agent-design.md), [plan](../superpowers/plans/2026-11-14-one-agent.md).

| Check | Result |
|---|---|
| `toolchain/check` | Exit 0: toolchain and hook tests, Worker tests, architecture and DOX, Go race/vet/format, 182 Flutter tests (14 new in `one_agent_test.dart`), analyze, format |
| Browser scenarios (chat, reminder, calendar, place lookup, email) | All pass. Email covers forwarded review, targeted answer, a visible failure with Retry, and a long pasted text with the same review, a source question, a quick reply and deletion |
| Real model, synthetic mail (`toolchain/real-model-agent-check`) | Pass: 11 calls, 0.0189 USD, turns: 1 arrival, 4 owner, 1 review reply. 13:00 start kept, no end or length, a how-long question, answers from the body and an attachment, only the answered candidate changed, same review from pasted text |
| APK 0.11.0+33 | arm64, 19,031,442 bytes, SHA-256 `68360a5f84ffd13ff8ac6c75ebf6c4bef484089b1336a7319211fedf376673a7`, `io.liveleds.minder`, code 2033, signer `7bbc3aba…d2572` (same as 0.10.3), full LAN download hash verified |

Not verified: production deployment, phone rendering, push display. The real-model run is one synthetic case, not a quality benchmark.
