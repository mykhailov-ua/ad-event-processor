import { apiConfirmed } from './confirmed_api.js';

/**
 * Create a self-serve API key for the current session user.
 *
 * @param {string} name
 * @returns {Promise<{ id: string, name: string, raw_key: string, expires_at?: string }>}
 */
export async function createApiKey(name) {
  const res = await apiConfirmed('/api/v1/selfserve/api-keys', {
    method: 'POST',
    body: JSON.stringify({ name }),
  });
  return res.data;
}
