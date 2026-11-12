const maxBytes = 20 * 1024 * 1024;
const encoder = new TextEncoder();

function mailbox(value) {
  const text = String(value ?? '').trim();
  const displayed = /^(?:"[^"\r\n]*"|[^<>"\r\n]*)<([^<>\r\n]+)>$/.exec(text);
  const address = (displayed ? displayed[1] : text).trim().toLowerCase();
  if (!/^[a-z0-9!#$%&'*+/=?^_`{|}~.-]+@[a-z0-9.-]+\.[a-z]{2,}$/i.test(address)) {
    throw new Error('Invalid email configuration or sender.');
  }
  const local = address.split('@')[0];
  if (local.startsWith('.') || local.endsWith('.') || local.includes('..')) {
    throw new Error('Invalid email configuration or sender.');
  }
  return address;
}

function settings(env) {
  const raw = String(env.EMAIL_ALLOWED_SENDERS ?? '');
  if (!raw.trim()) throw new Error('Email configuration is incomplete.');
  const allowed = raw.split(',').map(value => {
    const bare = value.trim();
    if (bare.includes('<') || bare.includes('>')) throw new Error('Use bare allowed addresses.');
    return mailbox(bare);
  });
  const destination = new URL(env.EMAIL_INGEST_URL);
  if (destination.protocol !== 'https:' || destination.username || destination.password || destination.hash) {
    throw new Error('Email ingestion requires a fixed HTTPS URL.');
  }
  const key = Uint8Array.from(atob(env.EMAIL_INGEST_HMAC_KEY), char => char.charCodeAt(0));
  if (key.length < 32 || !env.EMAIL_ACCESS_CLIENT_ID || !env.EMAIL_ACCESS_CLIENT_SECRET) {
    throw new Error('Email credentials are incomplete.');
  }
  return { allowed, recipient: mailbox(env.EMAIL_RECIPIENT), destination, key };
}

function authenticatedSender(message, config) {
  const from = mailbox(message.headers.get('from'));
  if (!config.allowed.includes(from) || mailbox(message.to) !== config.recipient) {
    throw new Error('Forwarding address is not allowed.');
  }
  // Cloudflare documents its first Authentication-Results header as the trusted verdict.
  // Never search later headers or the attached/quoted original for a passing result.
  const authentication = message.headers.get('authentication-results') ?? '';
  if (!/^\s*mx\.cloudflare\.net\s*;/i.test(authentication)) {
    throw new Error('Trusted sender authentication is missing.');
  }
  const dmarc = authentication.split(';').find(field => /^\s*dmarc=/i.test(field));
  const domain = from.slice(from.lastIndexOf('@') + 1);
  const authenticatedDomain = dmarc?.match(/\bheader\.from=([^\s;,]+)/i)?.[1]?.toLowerCase();
  if (!/^\s*dmarc=pass(?:\s|$)/i.test(dmarc ?? '') || authenticatedDomain !== domain) {
    throw new Error('Sender authentication failed.');
  }
  return from;
}

async function rawBytes(stream) {
  const reader = stream.getReader();
  const chunks = [];
  let length = 0;
  try {
    for (;;) {
      const { done, value } = await reader.read();
      if (done) break;
      length += value.byteLength;
      if (length > maxBytes) {
        await reader.cancel();
        throw new Error('Email exceeds the size limit.');
      }
      chunks.push(value);
    }
  } finally {
    reader.releaseLock();
  }
  const bytes = new Uint8Array(length);
  let offset = 0;
  for (const chunk of chunks) { bytes.set(chunk, offset); offset += chunk.byteLength; }
  return bytes;
}

const hex = bytes => Array.from(new Uint8Array(bytes), byte => byte.toString(16).padStart(2, '0')).join('');
function base64url(bytes) {
  return btoa(String.fromCharCode(...bytes)).replaceAll('+', '-').replaceAll('/', '_').replace(/=+$/, '');
}

export async function signedHeaders(config, envelope, bytes, timestamp, requestId) {
  const metadata = base64url(encoder.encode(JSON.stringify(envelope)));
  const digest = hex(await crypto.subtle.digest('SHA-256', bytes));
  const payload = ['v1', String(timestamp), requestId, metadata, digest].join('\n');
  const key = await crypto.subtle.importKey('raw', config.key, { name: 'HMAC', hash: 'SHA-256' }, false, ['sign']);
  const signature = hex(await crypto.subtle.sign('HMAC', key, encoder.encode(payload)));
  return {
    'Content-Type': 'message/rfc822',
    'CF-Access-Client-Id': config.clientId,
    'CF-Access-Client-Secret': config.clientSecret,
    'X-Minder-Email-Time': String(timestamp),
    'X-Minder-Email-ID': requestId,
    'X-Minder-Email-Envelope': metadata,
    'X-Minder-Email-Signature': signature,
  };
}

export async function receiveEmail(message, env, request = fetch) {
  let config, from, bytes;
  try {
    config = settings(env);
    from = authenticatedSender(message, config);
    if (message.rawSize > maxBytes) throw new Error('Email exceeds the size limit.');
    bytes = await rawBytes(message.raw);
  } catch {
    message.setReject('Email configuration, sender authentication or message size is invalid.');
    return;
  }
  const headers = await signedHeaders({
    key: config.key, clientId: env.EMAIL_ACCESS_CLIENT_ID, clientSecret: env.EMAIL_ACCESS_CLIENT_SECRET,
  }, { from, to: config.recipient, dmarc: 'pass' }, bytes, Math.floor(Date.now() / 1000), crypto.randomUUID());
  let response;
  try {
    response = await request(config.destination.href, {
      method: 'POST', body: bytes, headers, redirect: 'error', signal: AbortSignal.timeout(15000),
    });
  } catch {
    throw new Error('Minder did not acknowledge the email. Re-forward after checking the app.');
  }
  if (response.status !== 200 && response.status !== 201) {
    throw new Error('Minder rejected the email handoff.');
  }
  const result = await response.json();
  if (typeof result.capture_id !== 'string' || !result.capture_id) {
    throw new Error('Minder returned no capture receipt.');
  }
}

export default { email(message, env) { return receiveEmail(message, env); } };
