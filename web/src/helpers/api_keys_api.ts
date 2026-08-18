import { apiConfirmed } from './confirmed_api.js';

export type ApiKeyCreateResult = {
  id: string;
  name: string;
  raw_key: string;
  expires_at?: string;
};

export async function createApiKey(name: string): Promise<ApiKeyCreateResult> {
  const res = await apiConfirmed('/api/v1/selfserve/api-keys', {
    method: 'POST',
    body: JSON.stringify({ name }),
  });
  return res.data as ApiKeyCreateResult;
}
