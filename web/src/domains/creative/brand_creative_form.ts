export type ParseCreativeWeightResult = { ok: true; weight: number } | { ok: false; error: string };

const CREATIVE_WEIGHT_MAX = 2_147_483_647;

/** Strict integer parse; rejects parseInt slop like "100abc". Server: weight > 0. */
export function parseCreativeWeight(raw: string): ParseCreativeWeightResult {
  const trimmed = raw.trim();
  if (!trimmed) {
    return { ok: false, error: 'Weight is required.' };
  }
  if (!/^\d+$/.test(trimmed)) {
    return { ok: false, error: 'Weight must be a whole number.' };
  }
  const weight = Number(trimmed);
  if (!Number.isSafeInteger(weight)) {
    return { ok: false, error: 'Weight is out of range.' };
  }
  if (weight <= 0) {
    return { ok: false, error: 'Weight must be greater than zero.' };
  }
  if (weight > CREATIVE_WEIGHT_MAX) {
    return { ok: false, error: 'Weight is out of range.' };
  }
  return { ok: true, weight };
}

export type BuildBrandCreativeBodyResult =
  | {
      ok: true;
      body: { name: string; landing_url: string; weight: number; status: string };
    }
  | { ok: false; error: string };

export function buildBrandCreativeBody(
  nameRaw: string,
  landingUrlRaw: string,
  weightRaw: string,
  statusRaw: string
): BuildBrandCreativeBodyResult {
  const name = nameRaw.trim();
  if (!name) {
    return { ok: false, error: 'Name is required.' };
  }
  const landing_url = landingUrlRaw.trim();
  if (!landing_url) {
    return { ok: false, error: 'Landing URL is required.' };
  }
  const parsedWeight = parseCreativeWeight(weightRaw);
  if (!parsedWeight.ok) {
    return { ok: false, error: parsedWeight.error };
  }
  const status = statusRaw.trim() || 'active';
  return {
    ok: true,
    body: {
      name,
      landing_url,
      weight: parsedWeight.weight,
      status,
    },
  };
}
