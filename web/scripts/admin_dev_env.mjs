const API_TARGET = process.env.ADMIN_API_PROXY ?? 'http://127.0.0.1:8188';

export async function controlHealthy() {
  const healthUrl = new URL('/health', API_TARGET).toString();
  const ctrl = new AbortController();
  const timeout = setTimeout(() => ctrl.abort(), 2_000);
  try {
    const response = await fetch(healthUrl, { signal: ctrl.signal });
    return response.ok;
  } catch {
    return false;
  } finally {
    clearTimeout(timeout);
  }
}

export async function prepareDevBuildEnv() {
  if (process.env.ADMIN_DEV_AUTO_MOCK === undefined) {
    const healthy = await controlHealthy();
    process.env.ADMIN_DEV_AUTO_MOCK = healthy ? '0' : '1';
  }
  if (process.env.ADMIN_DEV_AUTO_MOCK === '1') {
    console.log(
      `Admin dev: control not reachable at ${API_TARGET}; mock API enabled (admin_dev).`,
    );
    console.log('Admin dev: force live API with ADMIN_DEV_AUTO_MOCK=0 or ?admin_dev=0');
  }
}

export function adminApiTarget() {
  return API_TARGET;
}
