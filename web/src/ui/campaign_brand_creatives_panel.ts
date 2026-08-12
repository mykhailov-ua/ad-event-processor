import { el, eventTargetValue } from '../lib/dom.js';
import { to } from '../lib/to.js';
import {
  createBrand,
  createBrandCreative,
  deleteBrandCreative,
  fetchBrandCreatives,
  updateBrandCreative,
} from '../helpers/brand_creatives_api.js';
import { mapServiceError } from '../helpers/service_error.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import { renderStatusBadge } from './status_badge.js';
import { tableSkeletonRows, renderEmptyTableCell } from './data_table.js';
import { renderButton } from './button.js';

export type BrandCreativeRow = {
  id: string;
  name: string;
  landing_url: string;
  weight: number;
  status: string;
};

export type CampaignBrandCreativesOpts = {
  brandId: string;
  customerId: string;
  canWrite: boolean;
  onBrandCreated: (id: string) => void;
};

export type CampaignBrandCreativesHandle = {
  destroy: () => void;
  reload: () => void;
};

/**
 * Narrow an API creative payload into a typed row.
 */
function asCreative(raw: unknown): BrandCreativeRow | null {
  if (!raw || typeof raw !== 'object') return null;
  const o = raw as Record<string, unknown>;
  if (typeof o.id !== 'string') return null;
  return {
    id: o.id,
    name: typeof o.name === 'string' ? o.name : '',
    landing_url: typeof o.landing_url === 'string' ? o.landing_url : '',
    weight: typeof o.weight === 'number' ? o.weight : Number(o.weight) || 0,
    status: typeof o.status === 'string' ? o.status : 'ACTIVE',
  };
}

/**
 * Mount weighted brand landing URLs CRUD for a campaign.
 */
export function mountCampaignBrandCreativesPanel(
  container: HTMLElement,
  opts: CampaignBrandCreativesOpts,
): CampaignBrandCreativesHandle {
  let destroyed = false;
  let brandId = opts.brandId;
  let creatives: BrandCreativeRow[] = [];
  let loading = true;
  let saving = false;
  let outboxHint: string | null = null;
  const form = {
    name: '',
    landing_url: '',
    weight: '100',
    status: 'ACTIVE',
  };

  function markOutboxQueued(action: string): void {
    outboxHint = `${action} queued to outbox — live on tracker typically within 60s (Redis brand creatives sync).`;
  }

  async function load(): Promise<void> {
    if (!brandId) {
      loading = false;
      creatives = [];
      render();
      return;
    }
    loading = true;
    render();
    const [rows, err] = await to(fetchBrandCreatives(brandId));
    if (destroyed) return;
    loading = false;
    creatives = err ? [] : (rows ?? []).map(asCreative).filter((c): c is BrandCreativeRow => c != null);
    render();
  }

  async function ensureBrand(): Promise<boolean> {
    if (brandId) return true;
    if (!opts.canWrite || !opts.customerId) return false;
    const [id, err] = await to(createBrand(opts.customerId, 'Default brand'));
    if (err) {
      pushToastMessage({ title: 'Brand create failed', message: mapServiceError(err).message });
      return false;
    }
    brandId = id ?? '';
    opts.onBrandCreated(brandId);
    return Boolean(brandId);
  }

  async function addCreative(): Promise<void> {
    if (!opts.canWrite || saving) return;
    saving = true;
    render();
    if (!(await ensureBrand())) {
      saving = false;
      render();
      return;
    }
    const body = {
      name: form.name.trim() || 'Landing',
      landing_url: form.landing_url.trim(),
      weight: Number.parseInt(form.weight, 10) || 100,
      status: form.status,
    };
    const [, err] = await to(createBrandCreative(brandId, body));
    saving = false;
    if (err) {
      if (err instanceof ConfirmCancelledError) { render(); return; }
      pushToastMessage({ title: 'Creative save failed', message: mapServiceError(err).message });
      render();
      return;
    }
    markOutboxQueued('Creative create');
    pushToastMessage({
      title: 'Creative saved',
      message: 'Hot-path sync queued via outbox',
    });
    form.name = '';
    form.landing_url = '';
    form.weight = '100';
    form.status = 'ACTIVE';
    void load();
  }

  async function togglePause(creative: BrandCreativeRow): Promise<void> {
    if (!opts.canWrite) return;
    const next = creative.status === 'PAUSED' ? 'ACTIVE' : 'PAUSED';
    const [, err] = await to(updateBrandCreative(creative.id, {
      name: creative.name,
      landing_url: creative.landing_url,
      weight: creative.weight,
      status: next,
    }));
    if (err) {
      if (err instanceof ConfirmCancelledError) return;
      pushToastMessage({ title: 'Update failed', message: mapServiceError(err).message });
      return;
    }
    markOutboxQueued(next === 'PAUSED' ? 'Creative pause' : 'Creative resume');
    void load();
  }

  async function removeCreative(id: string): Promise<void> {
    if (!opts.canWrite) return;
    const [, err] = await to(deleteBrandCreative(id));
    if (err) {
      if (err instanceof ConfirmCancelledError) return;
      pushToastMessage({ title: 'Delete failed', message: mapServiceError(err).message });
      return;
    }
    markOutboxQueued('Creative delete');
    void load();
  }

  function render(): void {
    container.replaceChildren(
      el('div', { className: 'stack', 'data-testid': 'campaign-brand-creatives' },
        el('p', { className: 'text-muted text-sm' },
          'Weighted landing URLs for click redirect rotation. Changes sync to Redis via outbox.',
        ),
        outboxHint
          ? el('p', {
            className: 'text-sm',
            'data-testid': 'creative-outbox-sync',
          },
            renderStatusBadge('pending', { kind: 'service', label: 'Outbox sync' }),
            ' ',
            outboxHint,
          )
          : null,
        !brandId && !opts.canWrite
          ? el('p', { className: 'text-muted' }, 'No brand linked to this campaign.')
          : null,
        brandId
          ? el('p', { className: 'text-hint text-sm' },
            'Brand: ',
            el('span', { className: 'font-mono' }, brandId),
          )
          : null,
        el('div', { className: 'table-wrapper' },
          el('table', { className: 'data-table' },
            el('thead', null,
              el('tr', null,
                el('th', null, 'Name'),
                el('th', null, 'Landing URL'),
                el('th', null, 'Weight'),
                el('th', null, 'Status'),
                el('th', null, ''),
              ),
            ),
            el('tbody', null,
              loading ? tableSkeletonRows(5) : null,
              !loading && creatives.length === 0
                ? el('tr', null,
                  renderEmptyTableCell(5, {
                    title: 'No creatives yet',
                    description: 'Add a landing URL and weight to start A/B rotation.',
                  }),
                )
                : null,
              creatives.map((c) => el('tr', null,
                el('td', null, c.name),
                el('td', { className: 'font-mono text-hint text-sm' }, c.landing_url),
                el('td', null, String(c.weight)),
                el('td', null, renderStatusBadge(c.status === 'ACTIVE' ? 'ACTIVE' : 'PAUSED')),
                el('td', null,
                  opts.canWrite
                    ? el('div', { className: 'flex gap-2' },
                      renderButton({
                        label: c.status === 'PAUSED' ? 'Resume' : 'Pause',
                        variant: 'secondary',
                        size: 'sm',
                        onClick: () => { void togglePause(c); },
                      }),
                      renderButton({
                        label: 'Delete',
                        variant: 'secondary',
                        size: 'sm',
                        onClick: () => { void removeCreative(c.id); },
                      }),
                    )
                    : null,
                ),
              )),
            ),
          ),
        ),
        opts.canWrite
          ? el('div', { className: 'section-card stack mt-4' },
            el('h3', { className: 'subsection-title' }, 'Add creative'),
            el('label', { className: 'form-field' }, 'Name',
              el('input', {
                className: 'form-input form-input--sm',
                value: form.name,
                onInput: (e: Event) => { form.name = eventTargetValue(e); },
              }),
            ),
            el('label', { className: 'form-field' }, 'Landing URL',
              el('input', {
                className: 'form-input',
                type: 'url',
                value: form.landing_url,
                onInput: (e: Event) => { form.landing_url = eventTargetValue(e); },
              }),
            ),
            el('div', { className: 'form-row' },
              el('label', { className: 'form-field' }, 'Weight',
                el('input', {
                  className: 'form-input form-input--sm',
                  inputMode: 'numeric',
                  value: form.weight,
                  onInput: (e: Event) => { form.weight = eventTargetValue(e); },
                }),
              ),
              el('label', { className: 'form-field' }, 'Status',
                el('select', {
                  className: 'form-select',
                  onChange: (e: Event) => { form.status = eventTargetValue(e); },
                },
                  el('option', { value: 'ACTIVE', selected: form.status === 'ACTIVE' }, 'Active'),
                  el('option', { value: 'PAUSED', selected: form.status === 'PAUSED' }, 'Paused'),
                ),
              ),
            ),
            renderButton({
              label: brandId ? 'Add creative' : 'Create brand & add',
              variant: 'primary',
              size: 'sm',
              loading: saving,
              disabled: saving || !form.landing_url.trim(),
              onClick: () => { void addCreative(); },
            }),
          )
          : null,
      ),
    );
  }

  void load();
  return {
    destroy() { destroyed = true; },
    reload: () => { void load(); },
  };
}
