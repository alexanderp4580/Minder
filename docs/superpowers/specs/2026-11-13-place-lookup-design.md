# Place lookup with Serper

Status: approved by the owner ("Implement", 2026-11-13). Inline execution, no sub-agents.

## Intent

The owner types "Aviad Landgraaf SnowWorld 17 to 24 november". Chat saves the title and dates but has no address or link for SnowWorld. A Serper places lookup supplies both. It sends text to an outside service, so the owner approves each query first.

## Goals

- When a request names a venue, chat offers a lookup and shows the exact query.
- The owner taps *Look it up* (or types it), which runs exactly one Serper places call. The next preview carries the address in `location` and the website in `website`.
- Tapping *Skip*, or any other reply, runs no call.
- With no `SERPER_API_KEY`, the feature is off: no offer, no call, chat unchanged.

## Non-goals and anti-goals

- No open web search and no event or date lookup. Dates and times stay as the owner wrote them.
- No migration, no new item field, no new route. The wire gains one additive, computed `options` field on chat entries; old clients show the offer as plain text.
- A query that runs without a visible offer and an explicit reply has failed. Query text in a log line has failed.

## Approaches

1. **Offer in chat with tap buttons and a deterministic server gate (chosen).** The offer carries two buttons, *Skip* and *Look it up*. A tap sends that text as an ordinary chat message. The server, not the model, decides whether to search.
2. Typed reply only ("Reply Look it up"). Rejected by the owner: no tap target.
3. A dedicated lookup card with Confirm and Discard. Rejected: a new entry type and route for a read-only call.

## Design

- **Offer.** The model calls `offer_lookup` (`query`, at most 100 characters) alone. The server stores a minder entry with fixed text: `Look up "<query>" on Google for its address and website?` The tool exists only when a place finder is configured.
- **Buttons.** The server marks an entry as an offer when its text has exactly that format and sets `options: ["Skip", "Look it up"]` on the wire. The value is computed when entries are served, not stored, so there is no migration. The app draws the offer as a card, like a preview, with the buttons inside it, and only for the newest chat entry. *Look it up* is the primary button. A tap calls the normal send path, so the choice appears in the history as your message.
- **Gate.** In `reply`, before any model call: the newest history entry, skipping error entries and earlier `Look it up` messages (so Retry works), is a minder entry in the offer format, and the owner's text, trimmed and case-folded with a trailing full stop removed, equals `look it up` (the tap sends exactly that, and typing it works the same). The server parses the query from that entry. The model never supplies the query or decides to search.
- **Search.** `server/serper` posts `{q, gl: "nl", hl: "en"}` to the places endpoint with a deadline and a bounded reply. It keeps at most three places (`title`, `address`, `category`, `website`, rating). It sends the key in a header and never logs it.
- **Result.** One model call follows, so there is no extra billed call. The messages gain the owner's text and the places as untrusted data. The model proposes the create or update with the address in `location` and the website in `website`. No match or a Serper error becomes data saying so; the model tells the owner and proposes the item without an address.
- **Skip.** Any other text goes through the normal flow. The prompt tells the model to propose the item without a lookup after a skipped offer, and never to offer twice for one request.
- **Config.** `SERPER_API_KEY` is optional. `SERPER_URL` may override the endpoint only outside production, like `OPENROUTER_URL`. Coolify gets the key as a process variable; deployment needs the owner's authorization.
- **Logs.** Info lines record `place_lookup` with query length, result count, status and `duration_ms`. Serper credits are counted in logs only; no euro accounting.

## Constraints

- Package order: `serper` is an adapter; `assistant` holds a `Places` interface and imports no HTTP client. `toolchain/architecture-check` enforces it.
- Tests use a local HTTP server for Serper and a fake model. No live Serper call in the test suite.
- The change reaches the app: `Entry.options`, a button row under the newest offer, tests and goldens, and a new APK (Telegram delivery is already authorized; phone approval gates merging).
- DOX: new `server/serper/AGENTS.md`; update `server/AGENTS.md`, `assistant`, `openrouter`, `config`, `cmd`, `api`, `app` (`lib`, `test`), `docs/mockups`, `README` (setting names and rules only) and `HANDOFF.md`.

## Acceptance

- "Aviad Landgraaf SnowWorld 17 to 24 november" produces an offer entry and zero Serper calls.
- Tapping *Look it up* makes one Serper call; the preview shows Witte Wereld 1 and the website.
- Tapping *Skip*, other text or answering a stale offer makes zero calls. A stale offer shows no buttons.
- Entries serve `options` only for a well-formed offer; other minder entries carry none.
- No key: no `offer_lookup` tool, no offer.
- Logs contain no query text or key. `toolchain/check` passes.

## Left to the builder

Exact prompt wording, the places schema fields beyond the ones above, and the test layout.
