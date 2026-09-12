import type { PlatformConfig, PlatformSettingsPatch } from '@/api/types';

export type PlatformSettingsDraft = {
  trackingDomain: string;
  defaultCurrency: string;
  timezone: string;
  ingressSchema: string;
  edgeXdp: boolean;
  edgeExposeClick: boolean;
  edgeExposeOpenrtb: boolean;
  stripeEnabled: boolean;
  stripeCheckoutSuccessUrl: string;
  stripeCheckoutCancelUrl: string;
  stripeSecretKey: string;
  stripeWebhookSecret: string;
};

export function normalizeTrackingDomainInput(value: string): string {
  let trimmed = value.trim();
  trimmed = trimmed.replace(/^https?:\/\//i, '');
  const slash = trimmed.indexOf('/');
  if (slash >= 0) {
    trimmed = trimmed.slice(0, slash);
  }
  return trimmed;
}

export function configToDraft(config: PlatformConfig | undefined): PlatformSettingsDraft {
  const stripe = config?.stripe;
  return {
    trackingDomain: config?.tracking_domain ?? '',
    defaultCurrency: config?.default_currency ?? '',
    timezone: config?.timezone ?? '',
    ingressSchema: config?.ingress_schema ?? '',
    edgeXdp: config?.edge_xdp ?? false,
    edgeExposeClick: config?.edge_expose_click ?? false,
    edgeExposeOpenrtb: config?.edge_expose_openrtb ?? false,
    stripeEnabled: stripe?.enabled ?? false,
    stripeCheckoutSuccessUrl: stripe?.checkout_success_url ?? '',
    stripeCheckoutCancelUrl: stripe?.checkout_cancel_url ?? '',
    stripeSecretKey: '',
    stripeWebhookSecret: '',
  };
}

function stripeConfigEquals(
  draft: PlatformSettingsDraft,
  config: PlatformConfig | undefined
): boolean {
  const stripe = config?.stripe;
  return (
    draft.stripeEnabled === (stripe?.enabled ?? false) &&
    draft.stripeCheckoutSuccessUrl === (stripe?.checkout_success_url ?? '') &&
    draft.stripeCheckoutCancelUrl === (stripe?.checkout_cancel_url ?? '') &&
    draft.stripeSecretKey === '' &&
    draft.stripeWebhookSecret === ''
  );
}

export function draftEqualsConfig(
  draft: PlatformSettingsDraft,
  config: PlatformConfig | undefined
): boolean {
  return (
    normalizeTrackingDomainInput(draft.trackingDomain) === (config?.tracking_domain ?? '') &&
    draft.defaultCurrency.trim().toUpperCase() ===
      (config?.default_currency ?? '').trim().toUpperCase() &&
    draft.timezone.trim() === (config?.timezone ?? '').trim() &&
    draft.ingressSchema === (config?.ingress_schema ?? '') &&
    draft.edgeXdp === (config?.edge_xdp ?? false) &&
    draft.edgeExposeClick === (config?.edge_expose_click ?? false) &&
    draft.edgeExposeOpenrtb === (config?.edge_expose_openrtb ?? false) &&
    stripeConfigEquals(draft, config)
  );
}

export function buildPlatformPatchBody(
  draft: PlatformSettingsDraft,
  snapshot: PlatformConfig | undefined
): PlatformSettingsPatch | null {
  if (!snapshot) {
    return null;
  }
  const patch: PlatformSettingsPatch = {};
  const normalizedDomain = normalizeTrackingDomainInput(draft.trackingDomain);
  if (normalizedDomain !== (snapshot.tracking_domain ?? '')) {
    patch.tracking_domain = normalizedDomain;
  }
  const currency = draft.defaultCurrency.trim().toUpperCase();
  if (currency !== (snapshot.default_currency ?? '').trim().toUpperCase()) {
    patch.default_currency = currency;
  }
  const timezone = draft.timezone.trim();
  if (timezone !== (snapshot.timezone ?? '').trim()) {
    patch.timezone = timezone;
  }
  if (draft.ingressSchema !== (snapshot.ingress_schema ?? '')) {
    patch.ingress_schema = draft.ingressSchema as PlatformSettingsPatch['ingress_schema'];
  }
  if (draft.edgeXdp !== (snapshot.edge_xdp ?? false)) {
    patch.edge_xdp = draft.edgeXdp;
  }
  if (draft.edgeExposeClick !== (snapshot.edge_expose_click ?? false)) {
    patch.edge_expose_click = draft.edgeExposeClick;
  }
  if (draft.edgeExposeOpenrtb !== (snapshot.edge_expose_openrtb ?? false)) {
    patch.edge_expose_openrtb = draft.edgeExposeOpenrtb;
  }

  const stripePatch: NonNullable<PlatformSettingsPatch['stripe']> = {};
  const snapshotStripe = snapshot.stripe;
  if (draft.stripeEnabled !== (snapshotStripe?.enabled ?? false)) {
    stripePatch.enabled = draft.stripeEnabled;
  }
  if (draft.stripeCheckoutSuccessUrl !== (snapshotStripe?.checkout_success_url ?? '')) {
    stripePatch.checkout_success_url = draft.stripeCheckoutSuccessUrl;
  }
  if (draft.stripeCheckoutCancelUrl !== (snapshotStripe?.checkout_cancel_url ?? '')) {
    stripePatch.checkout_cancel_url = draft.stripeCheckoutCancelUrl;
  }
  if (draft.stripeSecretKey.trim()) {
    stripePatch.secret_key = draft.stripeSecretKey.trim();
  }
  if (draft.stripeWebhookSecret.trim()) {
    stripePatch.webhook_secret = draft.stripeWebhookSecret.trim();
  }
  if (Object.keys(stripePatch).length > 0) {
    patch.stripe = stripePatch;
  }

  if (Object.keys(patch).length === 0) {
    return null;
  }
  return patch;
}
