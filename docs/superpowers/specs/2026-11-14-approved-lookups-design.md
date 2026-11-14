# Approved lookups design

Date: 2026-11-14. Replaces the lookup-loop design. The owner chose per-call approval: every search is shown first and runs only after a tap. The owner called a search budget useless and annoying, so there is none.

## Behaviour

- The model calls `offer_lookup(kind, query)` alone. `kind` is `maps`, `places` or `search`; the tool description states each kind's cost and returned fields. The server stores a fixed-format offer entry and serves it with the options `Skip` and `Look it up`.
- Offer texts: `Search Google Maps for "<query>"?`, `Search Google Places for "<query>"?`, `Search the web for "<query>"?`.
- Only when the owner's text is `Look it up` and the newest entry before it is such an offer does the server run that exact search through the `Finder`. The model chooses nothing at that point.
- The results return to the model as untrusted data in one extra user message, and the model gets its normal call: propose the item, ask the owner, or offer one more search (a new card, a new tap). A failed or empty search says so.
- Skip, any other text, a stale offer or no finder runs nothing. The model may offer again for any new owner message, including "try to get full address".
- The prompt never mentions a budget or a search count. It tells the model that `offer_lookup` is how it searches and that it should not say it cannot search.

## Data each kind returns (verified live, 2026-11-14)

| Kind | Endpoint | Credits | Returns |
|---|---|---|---|
| `maps` | `/maps` | 3 | up to three places: title, full address with postal code and town, phone, website, category, rating |
| `places` | `/places` | 1 | one best match: title, street address (postal code and town often missing), phone sometimes, website, category, rating |
| `search` | `/search` | 1 | up to three pages: title, link, snippet; no structured address or phone |

## Structure

- `domain` owns `LookupKind`, the offer format (`Offer{Kind, Query}`, `OfferText`, `ParseOffer`) and the computed quick replies. Nothing else reads offer text.
- `assistant` owns `Hit`, `Finder`, the kind descriptions, the gate that finds an approved offer, and the results message. It re-exports the kinds as aliases for the adapter.
- `serper` is unchanged: transport, endpoint table, decoders, client.
- `openrouter` builds `offer_lookup` with a `kind` enum from the assistant's descriptions. Tool sets follow `Message.Purpose`: chat, offer, email.
- No client change and no APK: the card shows server text and server buttons.

## Acceptance

- Tests first: offer format per kind and rejection of other text; the gate; the server runs the approved kind and query; the results message; skip and stale offers run nothing; an offer for each kind reaches the model; the prompt names no budget and says the model can search through `offer_lookup`; tool sets by purpose.
- `toolchain/check`, the place-lookup browser scenario, one live query per kind, and a real-model run of the conversation that failed on 2026-11-14 ("try to get full address" after an earlier lookup must produce an offer).
- Deploy only after the owner approves; rollback is `main` at `a8e5eb1`.
