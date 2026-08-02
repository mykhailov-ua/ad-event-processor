/**
 * Append a page to an immutable pages array without mutating the original.
 *
 * @param {Array<unknown>} pages
 * @param {unknown} page
 * @returns {Array<unknown>}
 */
export function appendPage(pages, page) {
  return [...pages, page];
}
