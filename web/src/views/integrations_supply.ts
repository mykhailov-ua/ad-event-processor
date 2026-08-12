import type { ViewHandle } from '../lib/router_types.js';
import { el, replaceChildren, eventTargetValue } from '../lib/dom.js';
import { to } from '../lib/to.js';
import * as auth from '../helpers/auth.js';
import { can } from '../helpers/permissions.js';
import { renderErrorBlock } from '../ui/error_block.js';
import { tableSkeletonRows } from '../ui/data_table.js';
import { renderButton } from '../ui/button.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { mapServiceError } from '../helpers/service_error.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import {
  createAdsTxtEntry,
  createSeller,
  deleteAdsTxtEntry,
  deleteSeller,
  fetchAdsTxtEntries,
  fetchAdsTxtPreview,
  fetchSellers,
  fetchSellersJSONPreview,
  fetchSupplyExportPath,
} from '../helpers/supply_api.js';

type SellerRow = {
  id: number;
  seller_id: string;
  domain: string;
  seller_type: string;
  name?: string;
  [key: string]: unknown;
};

type AdsTxtRow = {
  id: number;
  domain: string;
  publisher_account_id: string;
  relationship: string;
  [key: string]: unknown;
};

/**
 * Mount supply files admin (sellers.json + ads.txt).
 *
 * @param {HTMLElement} container
 * @returns {import('../lib/router.js').ViewHandle}
 */
export function mount(container: HTMLElement): ViewHandle {
  let destroyed = false;
  const canWrite = can(auth.getUser()?.permissions ?? [], 'settings:write');

  let tab = 'sellers';
  let sellers: SellerRow[] = [];
  let adsRows: AdsTxtRow[] = [];
  let exportPath = '';
  let sellersPreview = '';
  let adsPreview = '';
  let loading = true;
  let error: Error | string | null = null;
  let busy = false;

  const sellerForm = {
    seller_id: '',
    domain: '',
    seller_type: 'PUBLISHER',
    name: '',
    is_confidential: false,
  };
  const adsForm = {
    domain: '',
    publisher_account_id: '',
    relationship: 'DIRECT',
    cert_authority_id: '',
    sort_order: '0',
  };

  async function reload() {
    loading = true;
    error = null;
    render();
    const [sRes, aRes, pRes] = await Promise.all([
      to(fetchSellers()),
      to(fetchAdsTxtEntries()),
      to(fetchSupplyExportPath()),
    ]);
    if (destroyed) return;
    loading = false;
    if (sRes[1]) {
      error = sRes[1];
      render();
      return;
    }
    sellers = (sRes[0] ?? []) as SellerRow[];
    adsRows = aRes[1] ? [] : ((aRes[0] ?? []) as AdsTxtRow[]);
    exportPath = pRes[1] ? '' : (pRes[0] ?? '');
    render();
  }

  async function loadPreviews() {
    const [s, a] = await Promise.all([
      to(fetchSellersJSONPreview()),
      to(fetchAdsTxtPreview()),
    ]);
    if (destroyed) return;
    sellersPreview = s[1] ? `Error: ${s[1].message}` : (s[0] ?? '');
    adsPreview = a[1] ? `Error: ${a[1].message}` : (a[0] ?? '');
    render();
  }

  async function addSeller() {
    if (!canWrite) return;
    busy = true;
    render();
    const [, err] = await to(createSeller(sellerForm));
    busy = false;
    if (err) {
      if (err instanceof ConfirmCancelledError) { render(); return; }
      pushToastMessage({ title: 'Seller create failed', message: mapServiceError(err).message });
      render();
      return;
    }
    sellerForm.seller_id = '';
    sellerForm.domain = '';
    sellerForm.name = '';
    reload();
  }

  async function addAdsRow() {
    if (!canWrite) return;
    busy = true;
    render();
    const [, err] = await to(createAdsTxtEntry({
      domain: adsForm.domain.trim(),
      publisher_account_id: adsForm.publisher_account_id.trim(),
      relationship: adsForm.relationship.trim(),
      cert_authority_id: adsForm.cert_authority_id.trim(),
      sort_order: Number.parseInt(adsForm.sort_order, 10) || 0,
    }));
    busy = false;
    if (err) {
      if (err instanceof ConfirmCancelledError) { render(); return; }
      pushToastMessage({ title: 'ads.txt row failed', message: mapServiceError(err).message });
      render();
      return;
    }
    adsForm.domain = '';
    adsForm.publisher_account_id = '';
    reload();
  }

  function render() {
    if (destroyed) return;
    if (error) {
      replaceChildren(container, renderErrorBlock(error, 'Supply admin unavailable'));
      return;
    }

    replaceChildren(container,
      el('section', { className: 'stack', 'data-testid': 'supply-admin-view' },
        el('div', { className: 'page-header' },
          el('h1', { className: 'page-header__title' }, 'Supply files'),
          el('p', { className: 'page-header__desc' },
            'Manage sellers.json and ads.txt. Export path: ',
            el('code', { className: 'code-inline' }, exportPath || '—'),
          ),
        ),
        el('div', { className: 'filter-row cluster--actions' },
          renderButton({
            label: 'Sellers',
            variant: tab === 'sellers' ? 'primary' : 'secondary',
            size: 'sm',
            onClick: () => { tab = 'sellers'; render(); },
          }),
          renderButton({
            label: 'ads.txt',
            variant: tab === 'ads' ? 'primary' : 'secondary',
            size: 'sm',
            onClick: () => { tab = 'ads'; render(); },
          }),
          renderButton({
            label: 'Preview',
            variant: tab === 'preview' ? 'primary' : 'secondary',
            size: 'sm',
            onClick: () => { tab = 'preview'; loadPreviews(); },
          }),
        ),
        tab === 'sellers'
          ? el('div', { className: 'section-card stack' },
            el('div', { className: 'table-wrapper' },
              el('table', { className: 'data-table' },
                el('thead', null,
                  el('tr', null,
                    el('th', null, 'Seller ID'),
                    el('th', null, 'Domain'),
                    el('th', null, 'Type'),
                    el('th', null, 'Name'),
                    el('th', null, ''),
                  ),
                ),
                el('tbody', null,
                  loading ? tableSkeletonRows(5) : null,
                  sellers.map((s) => el('tr', null,
                    el('td', { className: 'font-mono' }, s.seller_id),
                    el('td', null, s.domain),
                    el('td', null, s.seller_type),
                    el('td', null, s.name || '—'),
                    el('td', null,
                      canWrite
                        ? renderButton({
                          label: 'Delete',
                          variant: 'secondary',
                          size: 'sm',
                          onClick: async () => {
                            const [, err] = await to(deleteSeller(s.id));
                            if (!err) reload();
                          },
                        })
                        : null,
                    ),
                  )),
                ),
              ),
            ),
            canWrite
              ? el('div', { className: 'stack mt-4' },
                el('h3', { className: 'subsection-title' }, 'Add seller'),
                el('div', { className: 'form-row' },
                  el('label', { className: 'form-field' }, 'Seller ID',
                    el('input', { className: 'form-input', value: sellerForm.seller_id, onInput: (e: Event) => { sellerForm.seller_id = eventTargetValue(e); } }),
                  ),
                  el('label', { className: 'form-field' }, 'Domain',
                    el('input', { className: 'form-input', value: sellerForm.domain, onInput: (e: Event) => { sellerForm.domain = eventTargetValue(e); } }),
                  ),
                ),
                el('div', { className: 'form-row' },
                  el('label', { className: 'form-field' }, 'Type',
                    el('select', {
                      className: 'form-select',
                      onChange: (e: Event) => { sellerForm.seller_type = eventTargetValue(e); },
                    },
                      el('option', { value: 'PUBLISHER' }, 'PUBLISHER'),
                      el('option', { value: 'INTERMEDIARY' }, 'INTERMEDIARY'),
                      el('option', { value: 'BOTH' }, 'BOTH'),
                    ),
                  ),
                  el('label', { className: 'form-field' }, 'Name',
                    el('input', { className: 'form-input', value: sellerForm.name, onInput: (e: Event) => { sellerForm.name = eventTargetValue(e); } }),
                  ),
                ),
                renderButton({
                  label: 'Add seller',
                  variant: 'primary',
                  size: 'sm',
                  loading: busy,
                  disabled: busy,
                  onClick: addSeller,
                }),
              )
              : null,
          )
          : null,
        tab === 'ads'
          ? el('div', { className: 'section-card stack' },
            el('div', { className: 'table-wrapper' },
              el('table', { className: 'data-table' },
                el('thead', null,
                  el('tr', null,
                    el('th', null, 'Domain'),
                    el('th', null, 'Account ID'),
                    el('th', null, 'Relationship'),
                    el('th', null, ''),
                  ),
                ),
                el('tbody', null,
                  loading ? tableSkeletonRows(4) : null,
                  adsRows.map((r) => el('tr', null,
                    el('td', null, r.domain),
                    el('td', { className: 'font-mono' }, r.publisher_account_id),
                    el('td', null, r.relationship),
                    el('td', null,
                      canWrite
                        ? renderButton({
                          label: 'Delete',
                          variant: 'secondary',
                          size: 'sm',
                          onClick: async () => {
                            const [, err] = await to(deleteAdsTxtEntry(r.id));
                            if (!err) reload();
                          },
                        })
                        : null,
                    ),
                  )),
                ),
              ),
            ),
            canWrite
              ? el('div', { className: 'stack mt-4' },
                el('h3', { className: 'subsection-title' }, 'Add ads.txt row'),
                el('div', { className: 'form-row' },
                  el('label', { className: 'form-field' }, 'Domain',
                    el('input', { className: 'form-input', value: adsForm.domain, onInput: (e: Event) => { adsForm.domain = eventTargetValue(e); } }),
                  ),
                  el('label', { className: 'form-field' }, 'Publisher account ID',
                    el('input', { className: 'form-input', value: adsForm.publisher_account_id, onInput: (e: Event) => { adsForm.publisher_account_id = eventTargetValue(e); } }),
                  ),
                  el('label', { className: 'form-field' }, 'Relationship',
                    el('select', {
                      className: 'form-select',
                      onChange: (e: Event) => { adsForm.relationship = eventTargetValue(e); },
                    },
                      el('option', { value: 'DIRECT' }, 'DIRECT'),
                      el('option', { value: 'RESELLER' }, 'RESELLER'),
                    ),
                  ),
                ),
                renderButton({
                  label: 'Add row',
                  variant: 'primary',
                  size: 'sm',
                  loading: busy,
                  disabled: busy,
                  onClick: addAdsRow,
                }),
              )
              : null,
          )
          : null,
        tab === 'preview'
          ? el('div', { className: 'section-card stack' },
            el('h3', { className: 'subsection-title' }, 'sellers.json'),
            el('pre', { className: 'code-block text-sm' }, sellersPreview || 'Loading…'),
            el('h3', { className: 'subsection-title' }, 'ads.txt'),
            el('pre', { className: 'code-block text-sm' }, adsPreview || 'Loading…'),
          )
          : null,
      ),
    );
  }

  reload();

  return {
    destroy() { destroyed = true; },
  };
}
