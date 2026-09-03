const UUID_RE =
  /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

const PLACEHOLDER_UUID_PREFIX = '00000000-0000-0000-0000-';

export function isUuidLike(value: string | undefined | null): boolean {
  if (!value?.trim()) {
    return false;
  }
  return UUID_RE.test(value.trim());
}

export function isPlaceholderSeedUuid(value: string | undefined | null): boolean {
  const trimmed = value?.trim().toLowerCase();
  if (!trimmed) {
    return false;
  }
  return trimmed.startsWith(PLACEHOLDER_UUID_PREFIX);
}

export function isHumanCustomerLabel(value: string | undefined | null): boolean {
  const trimmed = value?.trim();
  if (!trimmed) {
    return false;
  }
  return !isUuidLike(trimmed);
}

export function resolveCustomerLabel(
  customerId: string,
  customerNameById: Readonly<Record<string, string>>,
): string | undefined {
  const id = customerId.trim();
  if (!id) {
    return undefined;
  }
  const mapped = customerNameById[id]?.trim();
  if (isHumanCustomerLabel(mapped)) {
    return mapped;
  }
  return undefined;
}
