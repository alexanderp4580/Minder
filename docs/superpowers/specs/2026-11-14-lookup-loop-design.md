# Lookup loop design

Date: 2026-11-14. Extends the [place lookup design](2026-11-13-place-lookup-design.md). The owner asked for it on 2026-11-14: the agent chooses among Serper maps, places and search, judges the results, and asks the owner when unsure. No server fallbacks.

## Behaviour

- The offer card is unchanged: the model calls `offer_lookup` with a query, the owner sees Skip / Look it up. Skip and any other text run nothing.
- Look it up approves up to three Serper calls for that request. The server enforces the budget.
- After the tap the model receives three tools and chooses: `lookup_maps`, `lookup_places`, `lookup_search`, each with `query`. Results return as untrusted data. The model may call again, up to the budget, then must propose the item or ask the owner. A plain-text reply is a question to the owner.
- A reply that holds lookup calls drops its other calls; the model makes them again after the results.
- When the budget is spent the model gets no lookup tools. The result message says so.
- Prompt rule: use `lookup_maps` for an address, postal code or phone; `lookup_places` to confirm a venue cheaply; `lookup_search` for anything that is not one place. Never guess between plausible matches: ask the owner. Put the address in `location`, the phone in `phone`, the website in `website`.

## Data each tool returns (verified live, 2026-11-14)

| Tool | Endpoint | Credits | Returns |
|---|---|---|---|
| `lookup_maps` | `/maps` | 3 | up to three places: title, full address with postal code and town, phone, website, category, rating |
| `lookup_places` | `/places` | 1 | one best match: title, street address (postal code and town often missing), phone sometimes, website, category, rating |
| `lookup_search` | `/search` | 1 | up to three pages: title, link, snippet; no structured address or phone |

## Structure

- `assistant` owns the contract: `LookupKind`, `Hit`, `Finder`, the tool catalogue (name, description, kind), the budget and the loop. It has no HTTP.
- `serper` is split by responsibility: `transport.go` (one authenticated POST with a deadline and bounded body), `endpoint.go` (kind to path and decoder), `decode.go` (trim a provider body to `Hit`s), `client.go` (compose, log one line per call).
- `openrouter` builds the tool definitions from the assistant catalogue and picks the tool set from `Message.Purpose`; the `PlaceLookup` flag goes away.
- `SERPER_URL` becomes the base URL (`https://google.serper.dev`). Production accepts only that value.
- No client change and no APK: the offer card and entries are unchanged.

## Acceptance

- Tests first: transport, decoders per endpoint, endpoint table, client log line, config, budget (a fourth call never runs), ask-on-no-tool-call, dropped mixed calls, tool-set selection, the prompt rule.
- `toolchain/check`, the place-lookup browser scenario against the fixtures, one live query per tool.
- Deployed after approval; rollback `main` at `adb1bdf` or the previous `brighter-logo` commit `323e5fb`.
