import { apiConfirmed } from './confirmed_api.js';

export type ApiKeyCreateResult = {
  id: string;
  name: string;
  raw_key: string;
  expires_at?: string;
};

/**
 * Create a self-serve API key for the current session user.
 */
export async function createApiKey(name: string): Promise<ApiKeyCreateResult> {
  const res = await apiConfirmed('/api/v1/selfserve/api-keys', {
    method: 'POST',
    body: JSON.stringify({ name }),
  });
  return res.data as ApiKeyCreateResult;
}
