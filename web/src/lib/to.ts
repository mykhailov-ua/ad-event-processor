/**
 * Resolve a promise to a [data, err] tuple.
 */
export async function to<T>(promise: Promise<T>): Promise<[T, null] | [null, Error]> {
  try {
    return [await promise, null];
  } catch (err) {
    return [null, err instanceof Error ? err : new Error(String(err))];
  }
}
