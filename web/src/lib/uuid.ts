import { sha1 } from '@/lib/sha1';

/**
 * Shared UUID helpers for production code and dev-mock parity.
 *
 * Go parity:
 * - Namespace: uuid.NewSHA1(uuid.NameSpaceDNS, []byte("ad-event-processor.local.seed"))
 * - Deterministic row: seedDeterministicUUID(kind, seq) in cmd/admin/seed_catalog.go
 * - Runtime create: uuid.NewV7() / uuid.New() -- browser: newRandomUuid() (RFC 4122 v4)
 *
 * Verify:
 * node --import ./scripts/test_aliases.mjs --test --experimental-strip-types src/lib/uuid.test.ts
 * go test ./cmd/admin/ -short -run TestSeedCatalog_deterministicUUIDsAreRealistic -count=1
 */

// Matches cmd/admin/seed_catalog.go seedUUIDNamespace (UUID v5 over DNS namespace).
const SEED_UUID_NAMESPACE = 'f9ceb2a2-97a8-5602-afe5-076194dce015';

const UUID_RE = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

function uuidToBytes(uuid: string): Uint8Array {
  const hex = uuid.replace(/-/g, '');
  const bytes = new Uint8Array(16);
  for (let index = 0; index < 16; index += 1) {
    bytes[index] = Number.parseInt(hex.slice(index * 2, index * 2 + 2), 16);
  }
  return bytes;
}

function bytesToUuid(bytes: Uint8Array): string {
  const hex = Array.from(bytes, (byte) => byte.toString(16).padStart(2, '0')).join('');
  return `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(12, 16)}-${hex.slice(16, 20)}-${hex.slice(20)}`;
}

function uuidV5(name: string, namespace: string): string {
  const namespaceBytes = uuidToBytes(namespace);
  const nameBytes = new TextEncoder().encode(name);
  const payload = new Uint8Array(namespaceBytes.length + nameBytes.length);
  payload.set(namespaceBytes, 0);
  payload.set(nameBytes, namespaceBytes.length);
  const hash = sha1(payload);
  hash[6] = (hash[6] & 0x0f) | 0x50;
  hash[8] = (hash[8] & 0x3f) | 0x80;
  return bytesToUuid(hash.subarray(0, 16));
}

/** Deterministic seed id; parity with Go seedDeterministicUUID(entityKind, seq). */
export function seedDeterministicUuid(entityKind: string, seq: number): string {
  return uuidV5(`${entityKind}:${seq}`, SEED_UUID_NAMESPACE);
}

/** New runtime entity id (wizard session, commit, mutations). */
export function newRandomUuid(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID();
  }
  const bytes = new Uint8Array(16);
  if (typeof crypto !== 'undefined' && typeof crypto.getRandomValues === 'function') {
    crypto.getRandomValues(bytes);
  } else {
    for (let index = 0; index < bytes.length; index += 1) {
      bytes[index] = Math.floor(Math.random() * 256);
    }
  }
  bytes[6] = (bytes[6] & 0x0f) | 0x40;
  bytes[8] = (bytes[8] & 0x3f) | 0x80;
  return bytesToUuid(bytes);
}

export function isUuidLike(value: string | undefined | null): boolean {
  if (!value?.trim()) {
    return false;
  }
  return UUID_RE.test(value.trim());
}

/** Bans sequential / placeholder ids masquerading as UUIDs (not valid production or seed ids). */
export function isTrivialSequentialUuid(value: string | undefined | null): boolean {
  const trimmed = value?.trim().toLowerCase();
  if (!trimmed) {
    return false;
  }
  // Legacy non-hex type tags in old fixture ids (cust/camp) -- not valid RFC UUIDs.
  if (trimmed.startsWith('00000000-') && trimmed.includes('-4000-8000-')) {
    const tail = trimmed.slice(-12);
    if (/^0{9,}[0-9a-f]{0,3}$/.test(tail)) {
      return true;
    }
  }
  if (!UUID_RE.test(trimmed)) {
    return false;
  }
  if (trimmed.startsWith('00000000-0000-0000-0000-')) {
    return true;
  }
  const tail = trimmed.slice(-12);
  if (/^0{9,}[0-9a-f]{1,3}$/.test(tail)) {
    return true;
  }
  return false;
}
