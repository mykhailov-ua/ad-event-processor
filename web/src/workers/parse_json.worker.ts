/**
 * Parse a JSON string message and post the result (or null on failure).
 */
type ParseJsonWorkerSelf = {
  onmessage: ((ev: MessageEvent<unknown>) => void) | null;
  postMessage: (message: unknown) => void;
};

const parseJsonSelf = self as unknown as ParseJsonWorkerSelf;

parseJsonSelf.onmessage = (e: MessageEvent<unknown>) => {
  const raw = e.data;
  if (typeof raw !== 'string') {
    parseJsonSelf.postMessage(null);
    return;
  }
  try {
    parseJsonSelf.postMessage(JSON.parse(raw) as unknown);
  } catch {
    parseJsonSelf.postMessage(null);
  }
};

export {};
