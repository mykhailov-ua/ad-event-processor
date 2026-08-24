import { useCallback, useEffect, useState } from 'react';
import { ApiError } from '../helpers/api_client.js';
import * as auth from '../helpers/auth.js';
import { can } from '../helpers/permissions.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import { mapServiceError } from '../helpers/service_error.js';
import { displayLabel } from '../helpers/display_labels.js';
import { formatAmountMicro } from '../helpers/money.js';
import { createRtbDeal, deleteRtbDeal, fetchRtbDeals, patchRtbDeal } from '../helpers/rtb_api.js';
import type { RtbDealCreateSpec, RtbDealDTO } from '../types/rtb.js';
import { to } from '../lib/to.js';
import { useToast } from '../helpers/use_toast.js';
import { Breadcrumbs } from '../components/breadcrumbs.js';
import { Button } from '../components/button.js';
import { ErrorBlock } from '../components/error_block.js';
import { FormField } from '../components/form_field.js';
import { Modal } from '../components/modal.js';

export function RtbDealsPage() {
  const pushToast = useToast();
  const user = auth.getUser();
  const canWrite =
    can(user?.permissions ?? [], 'rtb:write') || can(user?.permissions ?? [], 'settings:write');

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<unknown | null>(null);
  const [deals, setDeals] = useState<RtbDealDTO[]>([]);
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<RtbDealDTO | null>(null);
  const [saving, setSaving] = useState(false);

  const [dealId, setDealId] = useState('');
  const [customerId, setCustomerId] = useState('');
  const [floorMicro, setFloorMicro] = useState('0');
  const [pacing, setPacing] = useState('even');
  const [seats, setSeats] = useState('1');

  const load = useCallback(async () => {
    setLoading(true);
    const [rows, err] = await to(fetchRtbDeals());
    setLoading(false);
    if (err) {
      setError(err);
      setDeals([]);
      return;
    }
    setError(null);
    setDeals(Array.isArray(rows) ? (rows as RtbDealDTO[]) : []);
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const openCreate = () => {
    setEditing(null);
    setDealId('');
    setCustomerId('');
    setFloorMicro('0');
    setPacing('even');
    setSeats('1');
    setModalOpen(true);
  };

  const openEdit = (deal: RtbDealDTO) => {
    setEditing(deal);
    setDealId(deal.deal_id);
    setCustomerId(deal.customer_id);
    setFloorMicro(String(deal.floor_micro ?? 0));
    setPacing(deal.pacing ?? 'even');
    setSeats(String(deal.seats ?? 1));
    setModalOpen(true);
  };

  const closeModal = () => {
    setModalOpen(false);
    setEditing(null);
  };

  const saveModal = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!canWrite || saving) return;
    const trimmedDealId = dealId.trim();
    const trimmedCustomer = customerId.trim();
    if (!trimmedDealId || !trimmedCustomer) {
      pushToast('Validation', 'deal_id and customer_id are required');
      return;
    }
    const spec: RtbDealCreateSpec = {
      deal_id: trimmedDealId,
      customer_id: trimmedCustomer,
      floor_micro: Number.parseInt(floorMicro, 10) || 0,
      pacing,
      seats: Number.parseInt(seats, 10) || 1,
    };
    setSaving(true);
    const [, err] = editing
      ? await to(patchRtbDeal(editing.id, spec))
      : await to(createRtbDeal(spec));
    setSaving(false);
    if (err) {
      if (err instanceof ConfirmCancelledError) return;
      const view = mapServiceError(err);
      pushToast(view.title, view.message, view.code);
      return;
    }
    pushToast('Deal saved', trimmedDealId);
    closeModal();
    void load();
  };

  const removeDeal = async (deal: RtbDealDTO) => {
    const [, err] = await to(deleteRtbDeal(deal.id));
    if (err) {
      if (err instanceof ConfirmCancelledError) return;
      const view = mapServiceError(err);
      pushToast(view.title, view.message, view.code);
      return;
    }
    pushToast('Deal deleted', deal.deal_id);
    void load();
  };

  if (loading && deals.length === 0 && !error) {
    return <p className="text-muted">Loading deals...</p>;
  }

  if (error && deals.length === 0) {
    const stub = error instanceof ApiError && (error.status === 404 || error.status === 501);
    if (!stub) {
      return <ErrorBlock error={error} fallbackTitle="Failed to load deals" />;
    }
  }

  return (
    <section className="stack" data-testid="rtb-deals-view">
      <div className="page-header">
        <Breadcrumbs items={[{ label: 'RTB', href: '/rtb/integration' }, { label: 'Deals' }]} />
        <div className="page-header__row">
          <h1 className="page-header__title">RTB deals</h1>
          {canWrite ? (
            <Button
              label="Create deal"
              variant="primary"
              size="sm"
              className="ml-auto"
              data-testid="rtb-deal-create-btn"
              onClick={openCreate}
            />
          ) : null}
        </div>
      </div>

      <div className="table-wrapper elevation-raised">
        <table className="data-table" data-testid="rtb-deals-table">
          <thead>
            <tr>
              <th scope="col">ID</th>
              <th scope="col">Deal ID</th>
              <th scope="col">Floor</th>
              <th scope="col">Customer</th>
              <th scope="col">Pacing</th>
              <th scope="col">Seats</th>
              <th scope="col">Updated</th>
              {canWrite ? <th scope="col" /> : null}
            </tr>
          </thead>
          <tbody>
            {deals.length === 0 ? (
              <tr>
                <td colSpan={canWrite ? 8 : 7} className="data-table__empty">
                  <div className="empty-state">
                    <div className="empty-state__title">No deals</div>
                    <div className="empty-state__desc text-muted text-sm">
                      Create a PMP deal to bind floor pricing to a customer.
                    </div>
                    {canWrite ? (
                      <Button
                        label="Create deal"
                        variant="secondary"
                        size="sm"
                        onClick={openCreate}
                      />
                    ) : null}
                  </div>
                </td>
              </tr>
            ) : null}
            {deals.map((deal) => (
              <tr key={deal.id}>
                <td>{String(deal.id)}</td>
                <td className="font-mono">{deal.deal_id}</td>
                <td className="font-mono">{formatAmountMicro(deal.floor_micro)}</td>
                <td className="font-mono text-xs">{deal.customer_id}</td>
                <td>{displayLabel(deal.pacing)}</td>
                <td>{String(deal.seats)}</td>
                <td className="text-muted text-xs">
                  {deal.updated_at ? new Date(deal.updated_at).toLocaleString() : '-'}
                </td>
                {canWrite ? (
                  <td>
                    <Button
                      label="Edit"
                      variant="secondary"
                      size="sm"
                      onClick={() => openEdit(deal)}
                    />{' '}
                    <Button
                      label="Delete"
                      variant="danger"
                      size="sm"
                      data-testid={`rtb-deal-delete-${deal.id}`}
                      onClick={() => removeDeal(deal)}
                    />
                  </td>
                ) : null}
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <Modal
        open={modalOpen}
        title={editing ? 'Edit deal' : 'Create deal'}
        onClose={closeModal}
        testId="rtb-deal-modal"
        actions={
          <>
            <Button
              label="Cancel"
              variant="secondary"
              size="sm"
              type="button"
              onClick={closeModal}
            />
            <Button
              label="Save"
              variant="primary"
              size="sm"
              type="submit"
              form="rtb-deal-form"
              loading={saving}
              disabled={saving}
            />
          </>
        }
      >
        <form id="rtb-deal-form" className="stack" onSubmit={(e) => void saveModal(e)}>
          <FormField label="Deal ID">
            <input
              id="deal-id"
              className="form-input"
              required
              value={dealId}
              disabled={Boolean(editing)}
              onChange={(e) => setDealId(e.target.value)}
            />
          </FormField>
          <FormField label="Customer ID">
            <input
              id="deal-customer"
              className="form-input"
              required
              value={customerId}
              onChange={(e) => setCustomerId(e.target.value)}
            />
          </FormField>
          <FormField label="Floor (micro)">
            <input
              id="deal-floor"
              className="form-input"
              type="number"
              min={0}
              value={floorMicro}
              onChange={(e) => setFloorMicro(e.target.value)}
            />
          </FormField>
          <FormField label="Pacing">
            <select
              id="deal-pacing"
              className="form-input form-input--sm"
              value={pacing}
              onChange={(e) => setPacing(e.target.value)}
            >
              <option value="even">Even</option>
              <option value="asap">ASAP</option>
            </select>
          </FormField>
          <FormField label="Seats">
            <input
              id="deal-seats"
              className="form-input form-input--sm"
              type="number"
              min={1}
              value={seats}
              onChange={(e) => setSeats(e.target.value)}
            />
          </FormField>
        </form>
      </Modal>
    </section>
  );
}
