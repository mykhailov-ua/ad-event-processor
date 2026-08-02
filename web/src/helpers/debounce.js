/**
 * Return a debounced wrapper that delays invoking fn until ms have elapsed.
 *
 * @param {() => void} fn
 * @param {number} ms
 * @returns {() => void}
 */
export function debounce(fn, ms) {
  let timer = null;
  return () => {
    if (timer) clearTimeout(timer);
    timer = setTimeout(fn, ms);
  };
}
