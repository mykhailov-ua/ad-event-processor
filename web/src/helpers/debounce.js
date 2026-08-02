/**
 * @param {() => void} fn
 * @param {number} ms
 */
export function debounce(fn, ms) {
  let timer = null;
  return () => {
    if (timer) clearTimeout(timer);
    timer = setTimeout(fn, ms);
  };
}
