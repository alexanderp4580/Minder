# Purpose

Cloudflare email admission and direct handoff.

# Ownership

Worker source, Node tests, deployment manifest and setup notes.

# Local Contracts

Use environment allowlists and secrets only. Authenticate the actual forwarder through Cloudflare's trusted first authentication result.
Preserve raw bytes, sign the envelope/body, require a durable capture receipt and never add content storage. Log one JSON line per stage (`event=email`, shared `id`, timings, counts, verdicts, status codes and failure reasons) and nothing else: no addresses, subjects, bodies, headers or credentials. New stages need a test that checks the log for leaks.

# Work Guidance

Use invented example-domain senders in tests. Do not deploy during local checks.

# Verification

`node --test email-worker/worker.test.mjs`.

# Child DOX Index

None.
