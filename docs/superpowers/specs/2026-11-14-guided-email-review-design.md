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
- When extraction finishes, one entry says `I found 3 appointments in "<subject>". We will go through them one at a time.` (`I found 1 appointment in "<subject>".` for one; `items` when they are not appointments).
- The server publishes one candidate at a time, in email order. A candidate that is not yet current has state `queued`; it has no preview and posts no question. The current candidate is announced (`Appointment 2 of 3: <title>`) and then shows either its question with quick replies or its preview with Confirm and Discard (Discard is the skip). Confirming or discarding publishes the next. When none remain: `Done: 2 added, 1 skipped.`
- The email card lists every appointment with its state (current, queued, confirmed, discarded). The flow lives in chat, where previews and questions already are.
- A question belongs to its candidate; only the current candidate can have an open question, so answering one never closes another.
- Every answer, quick reply or typed, goes to the model, which decides what it means. The code has no answer parser.
- Online meetings: the prompt tells the model an online meeting has no address, so it leaves `location` empty. The code has no list of venue words. Open in Google Maps shows whenever a location exists.
- Facts that fit no field (a meeting link, a dress code) go in `details`: an optional list of label and value pairs on an item. A proposal's `details` replaces the list; `clear_details` empties it. At most 10 pairs, label up to 60 characters, value up to 2,048. Previews and item details show them as `Label: value` rows; web links open.
- The model writes the opening message (`summary`) and the push line (`notification`) in its first extraction answer; the server posts and sends them unchanged. If either is missing the server posts no opening, and the push reads `An email needs your review.`

### Notifications
- The first push after extraction carries the model's `notification` text. The owner's own answers queue no push. A failure pushes a fixed sentence, `Minder could not read an email.`, because no model ran.

## Verification (acceptance)

- Tests first for every rule above: no automatic retry, failure entry and push, Retry without a cap, partial candidates, evidence reasons logged without content, the opening message counts, question closing per candidate, details round trip and validation, the model-written summary and notification, every answer reaching the model, no word list, no push after an answer, the 100,000-token request and the larger size limit.
- Flutter widget tests for the Maps rule and the card's candidate states; goldens only where existing goldens cover the card.
- `toolchain/check`; the email browser scenario extended with a three-appointment email; a real-model run on a synthetic three-appointment email with a Teams meeting (no real mailbox data).
- New APK (version bumped). Deploy after the owner approves.

## Left to the builder

Task order, file splits (keep one responsibility per file), exact fixed-sentence wording beyond the examples, how the stepper keeps its position across revisions.

## My calls (the owner can change these)

- The opening message comes after extraction, because the count is unknown before it. `Appointment i of N` and the `Done` summary stay mechanical progress notes.
- The step-by-step flow is driven by the server in chat, not by a stepper inside the email card, because previews, questions and Confirm/Discard already live in chat.
- A duration chip is applied locally without the model.
- Failures also push once, because the owner expected a pop-up.
- The email size limit goes to 400,000 bytes; I have not measured cost beyond the model's per-token price.
