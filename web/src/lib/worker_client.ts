export type WorkerHandle = {
  post: (data: unknown) => void;
  terminate: () => void;
};

export function spawnWorker(
  workerUrl: string | URL,
  onMessage: (data: unknown) => void
): WorkerHandle {
  const worker = new Worker(workerUrl, { type: 'module' });
  worker.onmessage = (event: MessageEvent) => onMessage(event.data);
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
