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
function capture(t) {
  const lines = [];
  for (const level of ['log', 'warn']) t.mock.method(console, level, (...args) => lines.push(JSON.parse(args[0])));
  return lines;
}
test('invalid addresses name the failing field only', async t => {
  const lines = capture(t);
  const mail = message(); mail.headers.set('from', 'not an address');
  await worker.email(mail, env, {});
  assert.deepEqual(lines.map(l => [l.stage, l.reason]), [['received', undefined], ['config_ok', undefined], ['rejected', 'Invalid From header address.']]);
});
test('quoted display names are accepted', async () => {
  const mail = message(); mail.headers.set('from', '"Owner, Test" <owner@example.com>');
  let calls = 0;
  await worker.email(mail, env, { fetch: async () => { calls++; return new Response(JSON.stringify({ capture_id: 'source' }), { status: 201 }); } });
  assert.equal(calls, 1); assert.equal(mail.rejection, undefined);
});
test('handoff failures log type, message and timing', async t => {
  const lines = capture(t);
  await assert.rejects(worker.email(message(), env, { fetch: async () => { throw new TypeError('redirect mode is set to error'); } }));
  const failed = lines.at(-1);
  assert.deepEqual([failed.stage, failed.error, failed.detail], ['handoff_failed', 'TypeError', 'redirect mode is set to error']);
  assert.equal(typeof failed.ms, 'number');
});
test('a successful email logs every stage and never any address, subject or secret', async t => {
  const lines = capture(t);
  await worker.email(message(), env, { fetch: async () => new Response(JSON.stringify({ capture_id: 'abc' }), { status: 201 }) });
  assert.deepEqual(lines.map(l => l.stage), ['received', 'config_ok', 'admitted', 'read', 'handoff', 'receipt']);
  assert.equal(lines[0].raw_size > 0, true);
  assert.equal(lines[2].dmarc, 'pass');
  assert.equal(lines[4].status, 201);
  assert.equal(new Set(lines.map(l => l.id)).size, 1);
  const text = JSON.stringify(lines);
  for (const secret of ['example.com', 'example.net', 'test-secret', 'test-id', 'Forwarded flight']) assert.equal(text.includes(secret), false, secret);
});
test('a rejected status logs the code and a missing receipt is flagged', async t => {
  const lines = capture(t);
  await assert.rejects(worker.email(message(), env, { fetch: async () => new Response('', { status: 302 }) }));
  assert.deepEqual([lines.at(-1).stage, lines.at(-1).status], ['handoff_rejected', 302]);
});
test('handoff uses a redirect mode the Workers runtime supports', async () => {
  let redirect;
  await worker.email(message(), env, { fetch: async (url, options) => { redirect = options.redirect; return new Response(JSON.stringify({ capture_id: 'x' }), { status: 201 }); } });
  assert.ok(['follow', 'manual'].includes(redirect), String(redirect));
});
