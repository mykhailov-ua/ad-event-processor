/**
 * @template T
 * @param {Promise<T>} promise
 * @returns {Promise<[T, null]|[null, Error]>}
 */
export async function to(promise) {
  try {
    return [await promise, null];
  } catch (err) {
    return [null, err instanceof Error ? err : new Error(String(err))];
  }
}
