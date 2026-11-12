# Purpose

Cloudflare email admission and direct handoff.

# Ownership

Worker source, Node tests, deployment manifest and setup notes.

# Local Contracts

Use environment allowlists and secrets only. Authenticate the actual forwarder through Cloudflare's trusted first authentication result.
Preserve raw bytes, sign the envelope/body, require a durable capture receipt and never add content storage or sensitive logs.

# Work Guidance

Use invented example-domain senders in tests. Do not deploy during local checks.

# Verification

`node --test email-worker/worker.test.mjs`.

# Child DOX Index

None.
