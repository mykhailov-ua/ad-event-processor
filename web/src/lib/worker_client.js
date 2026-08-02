/**
 * @param {string|URL} workerUrl
 * @param {(data: unknown) => void} onMessage
 */
export function spawnWorker(workerUrl, onMessage) {
  const worker = new Worker(workerUrl, { type: 'module' });
  worker.onmessage = (event) => onMessage(event.data);
  worker.onerror = () => {};
  return {
    post(data) {
      worker.postMessage(data);
    },
    terminate() {
      worker.terminate();
    },
  };
}
