import { isTrivialSequentialUuid, isUuidLike } from '@/api/dev_mock/seed_uuid';

export function isPlaceholderSeedUuid(value: string | undefined | null): boolean {
  return isTrivialSequentialUuid(value);
}

export { isUuidLike };

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
