import test from 'node:test';
import { readFileSync } from 'node:fs';
import assert from 'node:assert/strict';
import { receiveEmail, signedHeaders } from './worker.mjs';
const worker = { email: (mail, env, dependencies) => receiveEmail(mail, env, dependencies.fetch) };

const env = {
  EMAIL_ALLOWED_SENDERS: 'owner@example.com',
  EMAIL_RECIPIENT: 'assistant@example.net',
  EMAIL_INGEST_URL: 'https://minder.example.net/v1/email/ingest',
  EMAIL_INGEST_HMAC_KEY: Buffer.alloc(32, 7).toString('base64'),
  EMAIL_ACCESS_CLIENT_ID: 'test-id',
  EMAIL_ACCESS_CLIENT_SECRET: 'test-secret',
};
function message(from = 'owner@example.com', auth = 'mx.cloudflare.net; dmarc=pass header.from=example.com') {
  const bytes = Buffer.from('From: Owner <owner@example.com>\r\n\r\nForwarded flight');
  return {
    from, to: 'assistant@example.net', rawSize: bytes.length,
    headers: new Headers({ from: `Owner <${from}>`, 'authentication-results': auth }),
    raw: new Blob([bytes]).stream(),
    setReject(reason) { this.rejection = reason; },
  };
}
test('allowed authenticated forward sends exact raw bytes with a signature', async () => {
  const mail = message(); const seen = [];
  await worker.email(mail, env, { fetch: async (url, options) => {
    seen.push({ url, options });
    assert.equal(new TextDecoder().decode(options.body), 'From: Owner <owner@example.com>\r\n\r\nForwarded flight');
    assert.equal(options.headers['CF-Access-Client-Secret'], 'test-secret');
    assert.match(options.headers['X-Minder-Email-Signature'], /^[a-f0-9]{64}$/);
    return new Response(JSON.stringify({ capture_id: 'source' }), { status: 201 });
  }});
  assert.equal(seen.length, 1); assert.equal(mail.rejection, undefined);
});
test('unlisted forwarder and forged authentication never reach Minder', async () => {
  for (const mail of [message('other@example.com'), message('owner@example.com', 'attacker.example; dmarc=pass'), message('owner@example.com', 'mx.cloudflare.net; dmarc=fail')]) {
    let calls = 0;
    await worker.email(mail, env, { fetch: async () => { calls++; } });
    assert.equal(calls, 0); assert.ok(mail.rejection);
  }
});
test('missing allowlist and failed handoff cannot report success', async () => {
  const invalid = message(); await worker.email(invalid, { ...env, EMAIL_ALLOWED_SENDERS: '' }, {});
  assert.ok(invalid.rejection);
  await assert.rejects(worker.email(message(), env, { fetch: async () => new Response('', { status: 503 }) }));
});

test('Go and JavaScript use the same signed-envelope format', async () => {
 const headers = await signedHeaders({key: new Uint8Array(32).fill(7)}, {from:'owner@example.com',to:'assistant@example.net',dmarc:'pass'}, new TextEncoder().encode('From: Owner <owner@example.com>\r\nTo: assistant@example.net\r\n\r\nBooking'), 1794528000, 'request1');
 const key = await crypto.subtle.importKey('raw',new Uint8Array(32).fill(7),{name:'HMAC',hash:'SHA-256'},false,['verify']);
 const digest = Buffer.from(await crypto.subtle.digest('SHA-256',new TextEncoder().encode('From: Owner <owner@example.com>\r\nTo: assistant@example.net\r\n\r\nBooking'))).toString('hex');
 const payload = ['v1','1794528000','request1',headers['X-Minder-Email-Envelope'],digest].join('\n');
 assert.equal(await crypto.subtle.verify('HMAC',key,Buffer.from(headers['X-Minder-Email-Signature'],'hex'),new TextEncoder().encode(payload)),true);
});

test('signature matches the fixture verified by Go', async () => {
 const fixture=JSON.parse(readFileSync(new URL('../toolchain/email-ingress-vectors.json',import.meta.url)));
 const headers=await signedHeaders({key:Uint8Array.from(Buffer.from(fixture.keyBase64,'base64'))},fixture.envelope,new TextEncoder().encode(fixture.raw),fixture.timestamp,fixture.requestId);
 assert.equal(headers['X-Minder-Email-Signature'],fixture.headers['X-Minder-Email-Signature']);
 assert.equal(headers['X-Minder-Email-Envelope'],fixture.headers['X-Minder-Email-Envelope']);
});
