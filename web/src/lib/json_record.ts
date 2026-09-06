export type JsonRecord = Record<string, unknown>;

/** OpenAPI structs and API JSON objects accepted by JsonPayloadView. */
export type JsonPayload = JsonRecord | object;

export function isJsonRecord(value: unknown): value is JsonRecord {
  return value != null && typeof value === 'object' && !Array.isArray(value);
}

export function toJsonRecord(payload: JsonPayload): JsonRecord {
  return isJsonRecord(payload) ? payload : {};
}

export function readJsonString(record: JsonRecord, key: string): string | undefined {
  const value = record[key];
  return typeof value === 'string' ? value : undefined;
}
