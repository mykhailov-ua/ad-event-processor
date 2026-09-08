import type { ConsentRecord } from '@/api/types';

export function buildConsentRecordJson(body: ConsentRecord): string {
  const payload: ConsentRecord = {
    user_id: body.user_id,
    purposes: body.purposes,
    source: body.source,
  };
  const timestamp = body.timestamp?.trim();
  if (timestamp) {
    payload.timestamp = timestamp;
  }
  return JSON.stringify(payload);
}

export async function signConsentHmacHex(secret: string, bodyUtf8: string): Promise<string> {
  const key = await crypto.subtle.importKey(
    'raw',
    new TextEncoder().encode(secret),
    { name: 'HMAC', hash: 'SHA-256' },
    false,
    ['sign']
  );
  const signature = await crypto.subtle.sign('HMAC', key, new TextEncoder().encode(bodyUtf8));
  return [...new Uint8Array(signature)]
    .map((byte) => byte.toString(16).padStart(2, '0'))
    .join('');
}
