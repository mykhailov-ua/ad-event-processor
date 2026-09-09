const API_TARGET = process.env.ADMIN_API_PROXY ?? 'http://127.0.0.1:8188';

export function adminApiTarget() {
  return API_TARGET;
}

export async function prepareDevBuildEnv() {
  if (process.env.ADMIN_UI_BARE === undefined) {
    delete process.env.ADMIN_UI_BARE;
  }
}
