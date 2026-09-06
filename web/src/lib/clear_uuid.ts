/**
 * Clears optional UUID FK fields on campaign PATCH (Go uuid.Nil). Empty string decodes as 400.
 */
export const CLEAR_UUID = '00000000-0000-0000-0000-000000000000';

export function resolveOptionalUuidPatchValue(
  formValue: string,
  original: string | undefined
): string | undefined {
  const trimmed = formValue.trim();
  const previous = (original ?? '').trim();

  if (trimmed === previous) {
    return undefined;
  }

  if (trimmed === '') {
    return previous === '' ? undefined : CLEAR_UUID;
  }

  return trimmed;
}
