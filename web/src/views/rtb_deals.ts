import type { ViewHandle } from '../lib/router_types.js';
import type { RtbDealDTO, RtbDealCreateSpec } from '../types/api/rtb.js';
import { el, replaceChildren } from '../lib/dom.js';
import { to } from '../lib/to.js';
import { ApiError } from '../helpers/api_client.js';
import * as auth from '../helpers/auth.js';
import { can } from '../helpers/permissions.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import { mapServiceError } from '../helpers/service_error.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { renderErrorBlock } from '../ui/error_block.js';
import { renderBreadcrumbs } from '../ui/breadcrumbs.js';
import { formatAmountMicro } from '../helpers/money.js';
import { displayLabel } from '../helpers/display_labels.js';
import {
  createRtbDeal,
  deleteRtbDeal,
  fetchRtbDeals,
  patchRtbDeal,
} from '../helpers/rtb_api.js';
import { renderButton } from '../ui/button.js';
import { renderEmptyTableCell } from '../ui/data_table.js';

type DealsState = {
  loading: boolean;
  error: unknown | null;
  deals: RtbDealDTO[];
  modalOpen: boolean;
  editing: RtbDealDTO | null;
  saving: boolean;
};

/**
 * Mount RTB PMP deals CRUD view.
 */
export function mount(container: HTMLElement): ViewHandle {
  let destroyed = false;
  const state: DealsState = {
    loading: true,
    error: null,
    deals: [],
    modalOpen: false,
    editing: null,
    saving: false,
  };

  const user = auth.getUser();
  const canWrite = can(user?.permissions ?? [], 'rtb:write')
    || can(user?.permissions ?? [], 'settings:write');

  async function load(): Promise<void> {
    state.loading = true;
    render();
    const [rows, err] = await to(fetchRtbDeals());
    if (destroyed) return;
    state.loading = false;
    if (err) {
      state.error = err;
      state.deals = [];
    } else {
      state.error = null;
      state.deals = Array.isArray(rows) ? (rows as RtbDealDTO[]) : [];
    }
    render();
  }

  function openCreate(): void {
    state.editing = null;
    state.modalOpen = true;
    render();
  }

  function openEdit(deal: RtbDealDTO): void {
    state.editing = deal;
    state.modalOpen = true;
    render();
  }

  function closeModal(): void {
    state.modalOpen = false;
    state.editing = null;
    render();
  }

  async function saveModal(e: Event): Promise<void> {
    e.preventDefault();
    if (!canWrite || state.saving) return;
    const form = e.target as HTMLFormElement;
    const dealId = (form.querySelector('#deal-id') as HTMLInputElement)?.value.trim();
    const customerId = (form.querySelector('#deal-customer') as HTMLInputElement)?.value.trim();
    const floorMicro = Number.parseInt((form.querySelector('#deal-floor') as HTMLInputElement)?.value ?? '0', 10);
    const pacing = (form.querySelector('#deal-pacing') as HTMLSelectElement)?.value ?? 'even';
    const seats = Number.parseInt((form.querySelector('#deal-seats') as HTMLInputElement)?.value ?? '1', 10);
    if (!dealId || !customerId) {
      pushToastMessage({ title: 'Validation', message: 'deal_id and customer_id are required' });
      return;
    }
    const spec: RtbDealCreateSpec = {
      deal_id: dealId,
      customer_id: customerId,
      floor_micro: floorMicro,
      pacing,
      seats,
    };
    state.saving = true;
    render();
    const [, err] = state.editing
      ? await to(patchRtbDeal(state.editing.id, spec))
      : await to(createRtbDeal(spec));
    state.saving = false;
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
    pushToastMessage({ title: 'Deal saved', message: dealId });
    closeModal();
    load();
  }

  async function removeDeal(deal: RtbDealDTO): Promise<void> {
    const [, err] = await to(deleteRtbDeal(deal.id));
    if (err) {
      if (err instanceof ConfirmCancelledError) return;
      const view = mapServiceError(err);
      pushToastMessage({ title: view.title, message: view.message, code: view.code });
      return;
    }
    pushToastMessage({ title: 'Deal deleted', message: deal.deal_id });
    load();
  }

  function renderModal(): HTMLElement | null {
    if (!state.modalOpen) return null;
    const deal = state.editing;
    return el('dialog', {
      open: true,
      className: 'modal',
      'data-testid': 'rtb-deal-modal',
    },
      el('form', {
        className: 'stack p-4',
        onSubmit: saveModal,
      },
        el('h2', { className: 'subsection-title' }, deal ? 'Edit deal' : 'Create deal'),
        el('label', { className: 'form-field', htmlFor: 'deal-id' },
          'Deal ID',
          el('input', {
            id: 'deal-id',
            className: 'form-input',
            required: true,
            defaultValue: deal?.deal_id ?? '',
            disabled: Boolean(deal),
          }),
        ),
        el('label', { className: 'form-field', htmlFor: 'deal-customer' },
          'Customer ID',
          el('input', {
            id: 'deal-customer',
            className: 'form-input',
            required: true,
            defaultValue: deal?.customer_id ?? '',
          }),
        ),
        el('label', { className: 'form-field', htmlFor: 'deal-floor' },
          'Floor (micro)',
          el('input', {
            id: 'deal-floor',
            className: 'form-input',
            type: 'number',
            min: '0',
            defaultValue: String(deal?.floor_micro ?? 0),
          }),
        ),
        el('label', { className: 'form-field', htmlFor: 'deal-pacing' },
          'Pacing',
          el('select', {
            id: 'deal-pacing',
            className: 'form-input form-input--sm',
            defaultValue: deal?.pacing ?? 'even',
          },
            el('option', { value: 'even' }, 'Even'),
            el('option', { value: 'asap' }, 'ASAP'),
          ),
        ),
        el('label', { className: 'form-field', htmlFor: 'deal-seats' },
          'Seats',
          el('input', {
            id: 'deal-seats',
            className: 'form-input form-input--sm',
            type: 'number',
            min: '1',
            defaultValue: String(deal?.seats ?? 1),
          }),
        ),
        el('div', { className: 'flex gap-2 mt-2' },
          renderButton({
            label: 'Save',
            variant: 'primary',
            size: 'sm',
            type: 'submit',
            loading: state.saving,
            disabled: state.saving,
          }),
          renderButton({
            label: 'Cancel',
            variant: 'secondary',
            size: 'sm',
            onClick: closeModal,
          }),
        ),
      ),
    );
  }

  function render(): void {
    if (destroyed) return;

    if (state.loading && state.deals.length === 0) {
      replaceChildren(container, el('p', { className: 'text-muted' }, 'Loading deals…'));
      return;
    }

    if (state.error && state.deals.length === 0) {
      const stub = state.error instanceof ApiError && (state.error.status === 404 || state.error.status === 501);
      if (!stub) {
        replaceChildren(container, renderErrorBlock(state.error, 'Failed to load deals'));
        return;
      }
    }

    replaceChildren(container,
      el('section', { 'data-testid': 'rtb-deals-view' },
        el('div', { className: 'page-header' },
          renderBreadcrumbs([
            { label: 'RTB', href: '/rtb/integration' },
            { label: 'Deals' },
          ]),
          el('div', { className: 'page-header__row' },
            el('h1', { className: 'page-header__title' }, 'RTB deals'),
            canWrite
              ? renderButton({
                label: 'Create deal',
                variant: 'primary',
                size: 'sm',
                className: 'ml-auto',
                testId: 'rtb-deal-create-btn',
                onClick: openCreate,
              })
              : null,
          ),
        ),
        el('div', { className: 'table-wrapper elevation-raised' },
          el('table', { className: 'data-table', 'data-testid': 'rtb-deals-table' },
            el('thead', null,
              el('tr', null,
                el('th', { scope: 'col' }, 'ID'),
                el('th', { scope: 'col' }, 'Deal ID'),
                el('th', { scope: 'col' }, 'Floor'),
                el('th', { scope: 'col' }, 'Customer'),
                el('th', { scope: 'col' }, 'Pacing'),
                el('th', { scope: 'col' }, 'Seats'),
                el('th', { scope: 'col' }, 'Updated'),
                canWrite ? el('th', { scope: 'col' }, '') : null,
              ),
            ),
            el('tbody', null,
              state.deals.length === 0
                ? el('tr', null,
                  renderEmptyTableCell(canWrite ? 8 : 7, {
                    title: 'No deals',
                    description: 'Create a PMP deal to bind floor pricing to a customer.',
                    actionLabel: canWrite ? 'Create deal' : undefined,
                    onAction: canWrite ? openCreate : undefined,
                  }),
                )
                : null,
              state.deals.map((deal) => el('tr', null,
                el('td', null, String(deal.id)),
                el('td', { className: 'font-mono' }, deal.deal_id),
                el('td', { className: 'font-mono' }, formatAmountMicro(deal.floor_micro)),
                el('td', { className: 'font-mono text-xs' }, deal.customer_id),
                el('td', null, displayLabel(deal.pacing)),
                el('td', null, String(deal.seats)),
                el('td', { className: 'text-muted text-xs' },
                  deal.updated_at ? new Date(deal.updated_at).toLocaleString() : '—',
                ),
                canWrite
                  ? el('td', null,
                    renderButton({
                      label: 'Edit',
                      variant: 'secondary',
                      size: 'sm',
                      onClick: () => openEdit(deal),
                    }),
                    ' ',
                    renderButton({
                      label: 'Delete',
                      variant: 'danger',
                      size: 'sm',
                      testId: `rtb-deal-delete-${deal.id}`,
                      onClick: () => removeDeal(deal),
                    }),
                  )
                  : null,
              )),
            ),
          ),
        ),
        renderModal(),
      ),
    );
  }

  load();

  return {
    destroy() {
      destroyed = true;
    },
  };
}
