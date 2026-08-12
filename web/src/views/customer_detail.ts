import type { RouteContext, ViewHandle } from '../lib/router_types.js';
import type { CampaignDTO, CampaignListResponse } from '../types/api/campaign.js';
import type { CustomerDTO, TaxProfileDTO } from '../types/api/customer.js';
import type { WalletBalanceDTO } from '../types/api/billing.js';
import { el, replaceChildren } from '../lib/dom.js';
import { to } from '../lib/to.js';
import { apiBlob } from '../helpers/api_blob.js';
import { mapServiceError } from '../helpers/service_error.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { createResource, type ResourceState } from '../lib/fetch_resource.js';
import { renderErrorBlock } from '../ui/error_block.js';
import { clickableRow } from '../ui/clickable_row.js';
import { formatAmountMicro, formatUsdDecimal } from '../helpers/money.js';
import { ApiError } from '../helpers/api_client.js';
import { apiConfirmed } from '../helpers/confirmed_api.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import * as storage from '../helpers/storage.js';
import * as auth from '../helpers/auth.js';
import { can, isTenantUser } from '../helpers/permissions.js';
import { touchCustomerContext } from '../helpers/customer_context.js';
import { renderBreadcrumbs } from '../ui/breadcrumbs.js';
import { renderStatusBadge } from '../ui/status_badge.js';
import { renderIcon } from '../ui/icon.js';
import { mountCustomerApiKeysPanel } from './customer_api_keys_panel.js';
import {
  createSortState,
  toggleSort,
  sortRows,
  sortableTh,
  tableSkeletonRows,
  renderEmptyState,
} from '../ui/data_table.js';
import { renderButton, renderButtonLink } from '../ui/button.js';

/**
 * Mount the customer detail view with campaigns, wallet, and tax profile.
 *
 * @param {HTMLElement} container
 * @param {{ params: Record<string, string>, navigate: (path: string) => void }} ctx
 * @returns {import('../lib/router.js').ViewHandle}
 */
export function mount(container: HTMLElement, ctx: RouteContext): ViewHandle {
  let destroyed = false;
  const id = ctx.params.id;
  const user = auth.getUser();
  const tenant = isTenantUser(user?.role);
  const canWriteCustomer = can(user?.permissions ?? [], 'customers:write');
  const canCreateApiKey = can(user?.permissions ?? [], 'campaigns:write');

  const apiKeysSlot = el('div');
  /** @type {{ destroy: () => void }|null} */
  let apiKeysPanel: any = null;

  const customerState: ResourceState<CustomerDTO> = { data: null, loading: true, error: null };
  const campaignsState: ResourceState<CampaignListResponse> = { data: null, loading: true, error: null };
  const walletState: ResourceState<WalletBalanceDTO> = { data: null, loading: true, error: null };
  const taxState: ResourceState<TaxProfileDTO> = { data: null, loading: true, error: null };
  let taxSaving = false;
  let balanceExportBusy = false;
  const campaignSortState = createSortState('name', 'asc');

  touchCustomerContext(id);

  function openBilling() {
    if (!id) return;
    storage.setLastCustomerId(id);
    ctx.navigate(`/billing?customer_id=${encodeURIComponent(id)}`);
  }

  async function exportBalanceCsv(): Promise<void> {
    if (balanceExportBusy || !id) return;
    balanceExportBusy = true;
    render();
    const [blob, err] = await to(apiBlob(
      `/api/v1/customers/${encodeURIComponent(id)}/balance/export?format=csv`,
    ));
    balanceExportBusy = false;
    if (destroyed) return;
    if (err) {
      const view = mapServiceError(err);
      pushToastMessage({ title: view.title, message: view.message, code: view.code });
      render();
      return;
    }
    const url = URL.createObjectURL(blob);
    const anchor = document.createElement('a');
    anchor.href = url;
    anchor.download = `balance-${id.slice(0, 8)}.csv`;
    anchor.click();
    URL.revokeObjectURL(url);
    render();
  }

  async function saveTaxProfile(e: Event): Promise<void> {
    e.preventDefault();
    if (!canWriteCustomer || taxSaving) return;
    const form = e.target as HTMLFormElement;
    const body = {
      country_code: (form.querySelector('#tax-country') as HTMLInputElement)?.value.trim(),
      tax_region: (form.querySelector('#tax-region') as HTMLInputElement)?.value.trim(),
      tax_scheme: (form.querySelector('#tax-scheme') as HTMLInputElement)?.value.trim(),
      tax_rate_bps: Number.parseInt((form.querySelector('#tax-rate-bps') as HTMLInputElement)?.value ?? '0', 10),
    };
    taxSaving = true;
    render();
    const [, err] = await to(apiConfirmed(`/api/v1/customers/${id}/tax-profile`, {
      method: 'PUT',
      body: JSON.stringify(body),
    }));
    taxSaving = false;
    if (err) {
      if (err instanceof ConfirmCancelledError) {
        render();
        return;
      }
      const view = mapServiceError(err);
      pushToastMessage({ title: view.title, message: view.message, code: view.code });
      render();
      return;
    }
    pushToastMessage({ title: 'Tax profile saved', message: id });
    taxResource.reload();
  }

  if (tenant && user?.customer_id && id !== user.customer_id) {
    replaceChildren(container,
      renderErrorBlock(
        new ApiError(403, 'FORBIDDEN', 'tenant boundary'),
        'Access denied',
      ),
    );
    return {
      destroy() {
        destroyed = true;
      },
    };
  }

  function renderLoadingCards() {
    replaceChildren(container,
      el('div', { className: 'grid-stats section-block' },
        ['Name', 'Balance', 'Currency', 'Created'].map((label: any) =>
          el('div', { className: 'metric-card metric-card--loading' },
            el('div', { className: 'metric-card__label' }, label),
            el('div', { className: 'metric-card__value' }, '…'),
          ),
        ),
      ),
    );
  }

  function render() {
    if (destroyed) return;

    if (customerState.loading && !customerState.data) {
      renderLoadingCards();
      return;
    }

    if (customerState.error) {
      replaceChildren(container, renderErrorBlock(customerState.error, 'Failed to load customer'));
      return;
    }

    const customer = customerState.data;
    if (!customer) return;

    const campaigns = sortRows(campaignsState.data?.items ?? [], campaignSortState, {
      name: (c: CampaignDTO) => c.name ?? '',
      status: (c: CampaignDTO) => c.status ?? '',
      budget_limit: (c: CampaignDTO) => Number(c.budget_limit ?? 0),
    });

    const onCampaignSort = (key: string) => {
      toggleSort(campaignSortState, key);
      render();
    };

    replaceChildren(container,
      el('div', { className: 'page-header' },
        renderBreadcrumbs([
          { label: 'Customers', href: '/customers' },
          { label: customer.name ?? 'Customer' },
        ]),
        el('div', { className: 'page-header__row' },
          el('div', { className: 'flex items-center gap-2' },
            renderIcon('users', { size: 20, className: 'text-muted' }),
            el('h1', { className: 'page-header__title' }, customer.name),
          ),
          can(user?.permissions ?? [], 'customers:read')
            ? renderButton({
              label: balanceExportBusy ? 'Exporting…' : 'Export balance CSV',
              variant: 'secondary',
              size: 'sm',
              testId: 'customer-balance-export',
              loading: balanceExportBusy,
              disabled: balanceExportBusy,
              onClick: exportBalanceCsv,
            })
            : null,
        ),
      ),
      el('div', { className: 'grid-stats section-block' },
        el('div', { className: 'metric-card' },
          el('div', { className: 'metric-card__label' }, 'Balance'),
          el('div', { className: 'metric-card__value font-mono' }, formatUsdDecimal(customer.balance)),
        ),
        el('div', { className: 'metric-card' },
          el('div', { className: 'metric-card__label' }, 'Currency'),
          el('div', { className: 'metric-card__value' }, customer.currency ?? 'USD'),
        ),
        el('div', { className: 'metric-card' },
          el('div', { className: 'metric-card__label' }, 'Active campaigns'),
          el('div', { className: 'metric-card__value' }, String(customer.active_campaigns ?? 0)),
        ),
        el('div', { className: 'metric-card' },
          el('div', { className: 'metric-card__label' }, 'Total spend'),
          el('div', { className: 'metric-card__value font-mono' }, formatUsdDecimal(customer.total_spend)),
        ),
      ),
      el('section', { className: 'section-block' },
        el('h2', { className: 'subsection-title' }, 'Details'),
        el('dl', { className: 'definition-list' },
          el('dt', null, 'ID'),
          el('dd', { className: 'font-mono text-secondary' }, customer.id),
          el('dt', null, 'Created'),
          el('dd', null,
            customer.created_at ? new Date(customer.created_at).toLocaleString() : '—',
          ),
          el('dt', null, 'Updated'),
          el('dd', null,
            customer.updated_at ? new Date(customer.updated_at).toLocaleString() : '—',
          ),
        ),
      ),
      el('section', { className: 'section-block', 'data-testid': 'customer-tax-profile' },
        el('h2', { className: 'subsection-title' }, 'Tax profile'),
        taxState.loading ? el('span', { className: 'text-muted' }, 'Loading…') : null,
        taxState.error && !taxState.data
          ? el('p', { className: 'text-muted text-sm' }, 'Tax profile not available.')
          : null,
        canWriteCustomer
          ? el('form', { className: 'stack max-w-lg', onSubmit: saveTaxProfile },
            el('label', { className: 'form-field', htmlFor: 'tax-country' },
              'Country code',
              el('input', {
                id: 'tax-country',
                className: 'form-input form-input--sm',
                defaultValue: taxState.data?.country_code ?? '',
                required: true,
              }),
            ),
            el('label', { className: 'form-field', htmlFor: 'tax-region' },
              'Tax region',
              el('input', {
                id: 'tax-region',
                className: 'form-input form-input--sm',
                defaultValue: taxState.data?.tax_region ?? '',
              }),
            ),
            el('label', { className: 'form-field', htmlFor: 'tax-scheme' },
              'Scheme',
              el('input', {
                id: 'tax-scheme',
                className: 'form-input form-input--sm',
                defaultValue: taxState.data?.tax_scheme ?? 'standard',
                required: true,
              }),
            ),
            el('label', { className: 'form-field', htmlFor: 'tax-rate-bps' },
              'Rate (bps)',
              el('input', {
                id: 'tax-rate-bps',
                className: 'form-input form-input--sm',
                type: 'number',
                min: '0',
                defaultValue: String(taxState.data?.tax_rate_bps ?? 0),
              }),
            ),
            renderButton({
              label: taxSaving ? 'Saving…' : 'Save tax profile',
              variant: 'primary',
              size: 'sm',
              type: 'submit',
              loading: taxSaving,
              disabled: taxSaving,
            }),
          )
          : taxState.data
            ? el('dl', { className: 'definition-list' },
              el('dt', null, 'Country'),
              el('dd', null, taxState.data.country_code ?? '—'),
              el('dt', null, 'Region'),
              el('dd', null, taxState.data.tax_region ?? '—'),
              el('dt', null, 'Scheme'),
              el('dd', null, taxState.data.tax_scheme ?? '—'),
              el('dt', null, 'Rate (bps)'),
              el('dd', { className: 'font-mono' }, String(taxState.data.tax_rate_bps ?? 0)),
            )
            : (!taxState.loading && !taxState.error
              ? el('p', { className: 'text-muted text-sm' }, 'No tax profile on file.')
              : null),
      ),
      apiKeysSlot,
      el('section', { className: 'section-block' },
        el('div', { className: 'flex items-center gap-2 mb-3' },
          el('h2', { className: 'subsection-title' }, 'Wallet'),
          renderButton({
            label: 'Billing',
            variant: 'secondary',
            size: 'sm',
            onClick: openBilling,
          }),
        ),
        walletState.loading ? el('span', { className: 'text-muted' }, 'Loading…') : null,
        walletState.data
          ? el('div', { className: 'metric-row section-block' },
            el('div', { className: 'metric-card' },
              el('div', { className: 'metric-card__label' }, 'Balance (micro)'),
              el('div', { className: 'metric-card__value font-mono' },
                formatAmountMicro(walletState.data.balance_micro ?? 0, walletState.data.currency),
              ),
            ),
            el('div', { className: 'metric-card' },
              el('div', { className: 'metric-card__label' }, 'Overdraft'),
              el('div', { className: 'metric-card__value font-mono' },
                formatAmountMicro(walletState.data.allowed_overdraft_micro ?? 0, walletState.data.currency),
              ),
            ),
          )
          : null,
      ),
      el('section', { className: 'section-block' },
        el('div', { className: 'flex items-center gap-2 mb-3' },
          el('h2', { className: 'subsection-title' }, 'Campaigns'),
          renderButtonLink({
            label: 'All campaigns',
            href: `/campaigns?customer_id=${encodeURIComponent(id)}`,
            variant: 'secondary',
            size: 'sm',
          }),
        ),
        campaignsState.loading && campaigns.length === 0
          ? el('div', { className: 'table-wrapper elevation-raised' },
            el('table', { className: 'data-table' },
              el('tbody', null, tableSkeletonRows(3, 3)),
            ),
          )
          : null,
        !campaignsState.loading && campaigns.length === 0
          ? renderEmptyState({
            title: 'No campaigns',
            description: 'This customer has no campaigns yet.',
            actionLabel: 'View all campaigns',
            onAction: () => ctx.navigate(`/campaigns?customer_id=${encodeURIComponent(id)}`),
          })
          : null,
        !campaignsState.loading && campaigns.length > 0
          ? el('div', { className: 'table-wrapper elevation-raised' },
            el('table', { className: 'data-table' },
              el('thead', null,
                el('tr', null,
                  sortableTh('Name', 'name', campaignSortState, onCampaignSort),
                  sortableTh('Status', 'status', campaignSortState, onCampaignSort),
                  sortableTh('Budget', 'budget_limit', campaignSortState, onCampaignSort),
                ),
              ),
              el('tbody', null,
                campaigns.map((c: any) =>
                  clickableRow({
                    onActivate: () => ctx.navigate(`/campaigns/${c.id}`),
                    cells: [
                      el('td', { className: 'font-medium' }, c.name),
                      el('td', null, renderStatusBadge(c.status)),
                      el('td', { className: 'font-mono' }, formatUsdDecimal(c.budget_limit ?? '0.00')),
                    ],
                  }),
                ),
              ),
            ),
          )
          : null,
      ),
    );
    if (!apiKeysPanel && !destroyed) {
      apiKeysPanel = mountCustomerApiKeysPanel(apiKeysSlot, { canCreate: canCreateApiKey });
    }
  }

  const customerResource = createResource<CustomerDTO>(
    () => `/api/v1/customers/${id}`,
    {
      onUpdate: (s) => {
        Object.assign(customerState, s);
        render();
      },
    },
  );

  const campaignsResource = createResource<CampaignListResponse>(
    () => id
      ? `/api/v1/campaigns?customer_id=${encodeURIComponent(id)}&limit=10&offset=0`
      : null,
    {
      onUpdate: (s) => {
        Object.assign(campaignsState, s);
        render();
      },
    },
  );

  const walletResource = createResource<WalletBalanceDTO>(
    () => id ? `/api/v1/customers/${id}/wallet` : null,
    {
      onUpdate: (s) => {
        Object.assign(walletState, s);
        render();
      },
    },
  );

  const taxResource = createResource<TaxProfileDTO>(
    () => id ? `/api/v1/customers/${id}/tax-profile` : null,
    {
      onUpdate: (s) => {
        Object.assign(taxState, s);
        render();
      },
    },
  );

  render();

  return {
    destroy() {
      destroyed = true;
      apiKeysPanel?.destroy();
      customerResource.destroy();
      campaignsResource.destroy();
      walletResource.destroy();
      taxResource.destroy();
    },
  };
}
