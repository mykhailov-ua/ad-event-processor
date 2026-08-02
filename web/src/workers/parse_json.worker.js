self.onmessage = (e) => {
  try {
    self.postMessage(JSON.parse(e.data));
  } catch {
    self.postMessage(null);
  }
};
