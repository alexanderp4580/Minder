# Email ingress Worker

Receive only the configured forwarding addresses and recipient, then hand raw MIME to Minder over authenticated HTTPS.
Configure `EMAIL_ALLOWED_SENDERS`, `EMAIL_RECIPIENT`, `EMAIL_INGEST_URL`, `EMAIL_INGEST_HMAC_KEY`, `EMAIL_ACCESS_CLIENT_ID` and `EMAIL_ACCESS_CLIENT_SECRET` privately. The HMAC key is base64-encoded, at least 32 random bytes, and shared with Go. Do not commit actual sender values or secrets.

Run `node --test email-worker/worker.test.mjs` from the repository root.
Deploy using Wrangler after configuring the ingestion Access policy and private environment. Bind only the agreed subdomain/address in Email Routing. Keep root-domain mail records unchanged.

The Worker has no content buffer. A successful POST means Minder saved the source and extraction job. A failed handoff requires checking the app and re-forwarding; do not assume Cloudflare retries it.
Verify the trusted first `Authentication-Results` header from `mx.cloudflare.net` on a deployed test before live routing. Missing or failing sender authentication rejects capture.
