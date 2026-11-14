# Guided email review design

Date: 2026-11-14. Branch `guided-email-review`. Extends the [email capture design](2026-11-12-email-capture-design.md) and the [email review refinement](2026-11-13-email-review-refinement-design.md).

## Intent

On 2026-11-14 the owner forwarded an email with several appointments. Nothing appeared at first, because extraction failed and the server retried it silently. A retry then worked, but the review was hard to follow: "Teams" landed in the address field with an Open in Google Maps action, two duration questions sat in one chat message and both vanished when the owner answered one, and the follow-up produced a second "An email needs your review" push. The owner wants the assistant to guide them through an email, say what it found, never fail silently, and stop imposing limits.

## Goals

- A failed email always shows a message in the app, with the reason in plain words, and a Retry the owner taps. The server never retries by itself.
- No assistant-imposed limits on extraction: no call cap, an output ceiling of 100,000 tokens (the model allows 128,000), a larger email size limit. The €5 monthly spend limit stays.
- Several appointments are reviewed one at a time. The first message says how many were found.
- One bad quote or one unreadable appointment no longer loses the whole email.
- Online meetings leave the address empty; Maps appears only for a real address.
- Push notifications say what is true and never fire for the owner's own answer.

## Non-goals

- A summary card or a form for all appointments at once (the owner judged several-appointment emails rare and chose the simpler one-at-a-time flow).
- Changing how single-appointment emails look beyond the guided wording.
- Changing chat, lookups, the calendar or the monthly limit.

## Anti-goals

- A silent retry loop, a hidden cap, or a failure that shows nothing in the app.
- Replaying the whole email through the model for every chip tap.

## Constraints

- Extraction cites literal evidence per candidate; that validation stays.
- The email card, question entries and capture state machine exist; extend them rather than replace them.
- Secrets stay in TokenCrypt. English copy, short sentences.

## Approaches considered

1. **One summary card** listing every appointment with inline answers and a Confirm all. Best at a glance, more UI to build. Not chosen: more UI than a rare case needs.
2. **One appointment at a time (chosen by the owner).** The guided flow walks the candidates in order: a step shows the appointment and its question, if any, with Confirm and Skip. After each step it shows the next. Less UI, and it works for one appointment too.
3. **Leave the layout, fix only the bugs.** Cheapest, but the owner asked to be guided.

## Behaviour

### Failure and retries
- Extraction runs once per email revision, with the existing single in-call correction. Any failure ends the attempt: status `failed`, entry text `I could not read this email: <reason>. The original is kept.`, and one push `Minder could not read an email`.
- Reasons are short fixed sentences: the model's answer was cut off, the quoted evidence was missing, no appointment was found, the email was unreadable, the monthly limit was reached.
- Retry is the owner's button. There is no call cap and no automatic retry. Transient lease conflicts still release the claim without an attempt.
- Each rejected attempt logs a fixed reason code and counts, never email content.

### Limits
- `maxOutputTokens` becomes 100,000 for every model request (chat too). Verified: `anthropic/claude-haiku-5.5` allows 128,000 completion tokens and a 1,000,000-token context.
- The email source limit rises from 60,000 to 400,000 bytes; the chat request limits stay.

### Partial results
- Candidates with valid evidence are kept. Invalid ones are dropped and the review says `I could not verify N appointments from this email`, with the original available. If none are valid the attempt fails with the evidence reason.

### Guided review
- When extraction finishes, one entry says `I found 3 appointments in "<subject>". We will go through them one at a time.` (singular for one: `I found 1 appointment in "<subject>".`). Items that are not appointments are counted as `items`.
- The email card shows one candidate at a time with `Appointment 2 of 3`, its preview, its question and quick replies, and Confirm and Skip. Confirming or skipping moves to the next. The end step says `Done: 2 added, 1 skipped.`
- A question belongs to its candidate. Answering it closes that question only, never another candidate's.
- A quick reply that is a duration (`30 minutes`, `45 minutes`, `1 hour`, `90 minutes`, `1 hour 30 minutes`) is applied to the candidate directly with no model call. Other answers go through the model as today.
- Online meetings: the extraction prompt puts nothing in `location` and the join link, if any, in `website`; a note line says `- Online meeting (Teams)`. A server pass clears `location` when it names a virtual venue only (Teams, Zoom, Google Meet, Webex, online, video call, phone call).
- The app shows Open in Google Maps only for a location that is not a virtual venue.

### Notifications
- The first push after extraction reads `N appointments need your review` (or `1 appointment needs your review`). An answer by the owner queues no push. A failure pushes `Minder could not read an email`.

## Verification (acceptance)

- Tests first for every rule above: no automatic retry, failure entry and push, Retry without a cap, partial candidates, evidence reasons logged without content, the opening message counts, question closing per candidate, duration answer without a model call, virtual-venue clearing, Maps visibility, push texts, no push after an answer, the 100,000-token request and the larger size limit.
- Flutter widget tests for the stepper (steps, Skip, Confirm, done summary) and the Maps rule; goldens only where existing goldens cover the card.
- `toolchain/check`; the email browser scenario extended with a three-appointment email; a real-model run on a synthetic three-appointment email with a Teams meeting (no real mailbox data).
- New APK (version bumped). Deploy after the owner approves.

## Left to the builder

Task order, file splits (keep one responsibility per file), exact fixed-sentence wording beyond the examples, how the stepper keeps its position across revisions.

## My calls (the owner can change these)

- Opening message comes after extraction, because the count is unknown before it.
- A duration chip is applied locally without the model.
- Failures also push once, because the owner expected a pop-up.
- The email size limit goes to 400,000 bytes; I have not measured cost beyond the model's per-token price.
