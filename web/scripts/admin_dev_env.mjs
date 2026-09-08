const API_TARGET = process.env.ADMIN_API_PROXY ?? 'http://127.0.0.1:8188';

export function adminApiTarget() {
  return API_TARGET;
}

export async function prepareDevBuildEnv() {
  // No-op: local dev server always proxies /api/* to the control plane.
}
