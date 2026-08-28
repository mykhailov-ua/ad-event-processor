const LARGE_JSON_BYTES = 256 * 1024;

let parseWorker: Worker | null = null;

function parseJsonWorker(): Worker {
  if (!parseWorker) {
    parseWorker = new Worker('/src/workers/parse_json.worker.js', { type: 'module' });
  }
  return parseWorker;
}

export function parseJsonText(text: string): Promise<unknown> {
  if (text.length < LARGE_JSON_BYTES) {
    return Promise.resolve(JSON.parse(text) as unknown);
  }
  return new Promise((resolve, reject) => {
    const worker = parseJsonWorker();
    const onMessage = (event: MessageEvent<unknown>) => {
      worker.removeEventListener('message', onMessage);
      worker.removeEventListener('error', onError);
      resolve(event.data);
    };
    const onError = (event: ErrorEvent) => {
      worker.removeEventListener('message', onMessage);
      worker.removeEventListener('error', onError);
      reject(event.error ?? new Error('parse_json worker failed'));
    };
    worker.addEventListener('message', onMessage);
    worker.addEventListener('error', onError);
    worker.postMessage(text);
  });
}
