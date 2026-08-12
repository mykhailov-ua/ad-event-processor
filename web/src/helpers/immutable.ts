/**
 * Append a page to an immutable pages array without mutating the original.
 */
export function appendPage(pages: unknown[], page: unknown): unknown[] {
  return [...pages, page];
}
