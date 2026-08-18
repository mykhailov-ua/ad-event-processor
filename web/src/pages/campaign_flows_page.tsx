import { useCallback, useEffect, useState } from 'react';
import { to } from '../lib/to.js';
import * as auth from '../helpers/auth.js';
import { can } from '../helpers/permissions.js';
import { mapServiceError } from '../helpers/service_error.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import {
  createFlow,
  createLander,
  createOffer,
  fetchFlows,
  fetchLanders,
  fetchOffers,
  parseFlowPaths,
  summarizeFlowPaths,
  type FlowDTO,
  type LanderDTO,
  type OfferDTO,
} from '../helpers/flows_api.js';
import { Breadcrumbs } from '../components/breadcrumbs.js';
import { Button } from '../components/button.js';
import { CopyableUuid } from '../components/copyable_uuid.js';
import { ErrorBlock } from '../components/error_block.js';
import { TabBar } from '../components/tab_bar.js';

type FlowTab = 'landers' | 'offers' | 'flows';

function TableSkeleton({ cols, rows = 4 }: { cols: number; rows?: number }) {
  return (
    <>
      {Array.from({ length: rows }, (_, i) => (
        <tr key={`sk-${i}`} className="data-table__row--skeleton" aria-hidden="true">
          {Array.from({ length: cols }, (__, j) => (
            <td key={`sk-${i}-${j}`}>
              <span className="skeleton-bar" />
            </td>
          ))}
        </tr>
      ))}
    </>
  );
}

/**
 * Landers, offers, and weighted flow paths (list-based; no visual builder).
 */
export function CampaignFlowsPage() {
  const canWrite = can(auth.getUser()?.permissions ?? [], 'campaigns:write');
  const [tab, setTab] = useState<FlowTab>('landers');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<unknown>(null);
  const [busy, setBusy] = useState(false);

  const [landers, setLanders] = useState<LanderDTO[]>([]);
  const [offers, setOffers] = useState<OfferDTO[]>([]);
  const [flows, setFlows] = useState<FlowDTO[]>([]);

  const [landerForm, setLanderForm] = useState({ name: '', url: '' });
  const [offerForm, setOfferForm] = useState({ name: '', url: '' });
  const [flowForm, setFlowForm] = useState({
    name: '',
    landerId: '',
    landerWeight: '100',
    offerId: '',
    offerWeight: '100',
  });

  const reload = useCallback(async () => {
    setLoading(true);
    setError(null);
    const [lRes, oRes, fRes] = await Promise.all([
      to(fetchLanders()),
      to(fetchOffers()),
      to(fetchFlows()),
    ]);
    setLoading(false);
    if (lRes[1]) {
      setError(lRes[1]);
      return;
    }
    setLanders(lRes[0] ?? []);
    setOffers(oRes[1] ? [] : (oRes[0] ?? []));
    setFlows(fRes[1] ? [] : (fRes[0] ?? []));
  }, []);

  useEffect(() => {
    void reload();
  }, [reload]);

  const submitLander = async () => {
    if (!canWrite || busy) return;
    const name = landerForm.name.trim();
    const url = landerForm.url.trim();
    if (!name || !url) {
      pushToastMessage({ title: 'Missing fields', message: 'Name and URL are required' });
      return;
    }
    if (!/^https?:\/\//i.test(url)) {
      pushToastMessage({
        title: 'Invalid URL',
        message: 'URL must start with http:// or https://',
      });
      return;
    }
    setBusy(true);
    const [, err] = await to(createLander(name, url));
    setBusy(false);
    if (err) {
      if (err instanceof ConfirmCancelledError) return;
      pushToastMessage({ title: 'Create failed', message: mapServiceError(err).message });
      return;
    }
    setLanderForm({ name: '', url: '' });
    pushToastMessage({ title: 'Lander created', message: name });
    await reload();
  };

  const submitOffer = async () => {
    if (!canWrite || busy) return;
    const name = offerForm.name.trim();
    const url = offerForm.url.trim();
    if (!name || !url) {
      pushToastMessage({ title: 'Missing fields', message: 'Name and URL are required' });
      return;
    }
    if (!/^https?:\/\//i.test(url)) {
      pushToastMessage({
        title: 'Invalid URL',
        message: 'URL must start with http:// or https://',
      });
      return;
    }
    setBusy(true);
    const [, err] = await to(createOffer(name, url));
    setBusy(false);
    if (err) {
      if (err instanceof ConfirmCancelledError) return;
      pushToastMessage({ title: 'Create failed', message: mapServiceError(err).message });
      return;
    }
    setOfferForm({ name: '', url: '' });
    pushToastMessage({ title: 'Offer created', message: name });
    await reload();
  };

  const submitFlow = async () => {
    if (!canWrite || busy) return;
    const name = flowForm.name.trim();
    const landerWeight = Number.parseInt(flowForm.landerWeight, 10);
    const offerWeight = Number.parseInt(flowForm.offerWeight, 10);
    if (!name || !flowForm.landerId || !flowForm.offerId) {
      pushToastMessage({
        title: 'Missing fields',
        message: 'Name, lander, and offer are required',
      });
      return;
    }
    if (
      !Number.isFinite(landerWeight) ||
      landerWeight <= 0 ||
      !Number.isFinite(offerWeight) ||
      offerWeight <= 0
    ) {
      pushToastMessage({ title: 'Invalid weights', message: 'Weights must be positive integers' });
      return;
    }
    setBusy(true);
    const [, err] = await to(
      createFlow(name, [
        {
          weight: 100,
          landers: [{ lander_id: flowForm.landerId, weight: landerWeight }],
          offers: [{ offer_id: flowForm.offerId, weight: offerWeight }],
        },
      ])
    );
    setBusy(false);
    if (err) {
      if (err instanceof ConfirmCancelledError) return;
      pushToastMessage({ title: 'Create failed', message: mapServiceError(err).message });
      return;
    }
    setFlowForm({ name: '', landerId: '', landerWeight: '100', offerId: '', offerWeight: '100' });
    pushToastMessage({ title: 'Flow created', message: name });
    await reload();
  };

  if (error) {
    return <ErrorBlock error={error} />;
  }

  return (
    <>
      <div className="page-header">
        <Breadcrumbs items={[{ label: 'Campaigns', href: '/campaigns' }, { label: 'Flows' }]} />
        <h1 className="page-header__title">Landers, offers &amp; flows</h1>
        <p className="text-muted text-sm">
          Declarative flow routing for /click — weighted lander and offer selection per path.
        </p>
      </div>

      <TabBar
        tabs={[
          { id: 'landers', label: 'Landers' },
          { id: 'offers', label: 'Offers' },
          { id: 'flows', label: 'Flows' },
        ]}
        active={tab}
        onChange={(id) => setTab(id as FlowTab)}
      />

      {tab === 'landers' ? (
        <div className="section-block stack">
          {canWrite ? (
            <section className="section-card stack" data-testid="flow-lander-form">
              <h3 className="subsection-title">Add lander</h3>
              <label className="form-field" htmlFor="lander-name">
                Name
                <input
                  id="lander-name"
                  className="form-input form-input--sm"
                  value={landerForm.name}
                  onChange={(e) => setLanderForm((f) => ({ ...f, name: e.target.value }))}
                />
              </label>
              <label className="form-field" htmlFor="lander-url">
                Landing URL
                <input
                  id="lander-url"
                  className="form-input form-input--sm"
                  value={landerForm.url}
                  placeholder="https://…"
                  onChange={(e) => setLanderForm((f) => ({ ...f, url: e.target.value }))}
                />
              </label>
              <Button
                label={busy ? 'Saving…' : 'Create lander'}
                variant="primary"
                size="sm"
                loading={busy}
                disabled={busy}
                data-testid="flow-lander-submit"
                onClick={() => void submitLander()}
              />
            </section>
          ) : null}
          <section className="section-card">
            <table className="data-table" data-testid="flow-landers-table">
              <thead>
                <tr>
                  <th>Name</th>
                  <th>URL</th>
                  <th>ID</th>
                </tr>
              </thead>
              <tbody>
                {loading ? <TableSkeleton cols={3} /> : null}
                {!loading && landers.length === 0 ? (
                  <tr>
                    <td colSpan={3} className="text-muted">
                      No landers yet
                    </td>
                  </tr>
                ) : null}
                {!loading
                  ? landers.map((row) => (
                      <tr key={row.id}>
                        <td>{row.name}</td>
                        <td className="font-mono text-sm">{row.url}</td>
                        <td>
                          <CopyableUuid uuid={row.id} />
                        </td>
                      </tr>
                    ))
                  : null}
              </tbody>
            </table>
          </section>
        </div>
      ) : null}

      {tab === 'offers' ? (
        <div className="section-block stack">
          {canWrite ? (
            <section className="section-card stack" data-testid="flow-offer-form">
              <h3 className="subsection-title">Add offer</h3>
              <label className="form-field" htmlFor="offer-name">
                Name
                <input
                  id="offer-name"
                  className="form-input form-input--sm"
                  value={offerForm.name}
                  onChange={(e) => setOfferForm((f) => ({ ...f, name: e.target.value }))}
                />
              </label>
              <label className="form-field" htmlFor="offer-url">
                Offer URL
                <input
                  id="offer-url"
                  className="form-input form-input--sm"
                  value={offerForm.url}
                  placeholder="https://…"
                  onChange={(e) => setOfferForm((f) => ({ ...f, url: e.target.value }))}
                />
              </label>
              <Button
                label={busy ? 'Saving…' : 'Create offer'}
                variant="primary"
                size="sm"
                loading={busy}
                disabled={busy}
                data-testid="flow-offer-submit"
                onClick={() => void submitOffer()}
              />
            </section>
          ) : null}
          <section className="section-card">
            <table className="data-table" data-testid="flow-offers-table">
              <thead>
                <tr>
                  <th>Name</th>
                  <th>URL</th>
                  <th>ID</th>
                </tr>
              </thead>
              <tbody>
                {loading ? <TableSkeleton cols={3} /> : null}
                {!loading && offers.length === 0 ? (
                  <tr>
                    <td colSpan={3} className="text-muted">
                      No offers yet
                    </td>
                  </tr>
                ) : null}
                {!loading
                  ? offers.map((row) => (
                      <tr key={row.id}>
                        <td>{row.name}</td>
                        <td className="font-mono text-sm">{row.url}</td>
                        <td>
                          <CopyableUuid uuid={row.id} />
                        </td>
                      </tr>
                    ))
                  : null}
              </tbody>
            </table>
          </section>
        </div>
      ) : null}

      {tab === 'flows' ? (
        <div className="section-block stack">
          {canWrite ? (
            <section className="section-card stack" data-testid="flow-create-form">
              <h3 className="subsection-title">Create flow</h3>
              <p className="text-muted text-sm">
                Single-path flow with one lander and one offer. Add more landers/offers first, then
                combine here.
              </p>
              <label className="form-field" htmlFor="flow-name">
                Flow name
                <input
                  id="flow-name"
                  className="form-input form-input--sm"
                  value={flowForm.name}
                  onChange={(e) => setFlowForm((f) => ({ ...f, name: e.target.value }))}
                />
              </label>
              <div className="grid-2">
                <label className="form-field" htmlFor="flow-lander">
                  Lander
                  <select
                    id="flow-lander"
                    className="form-input form-input--sm"
                    value={flowForm.landerId}
                    onChange={(e) => setFlowForm((f) => ({ ...f, landerId: e.target.value }))}
                  >
                    <option value="">Select…</option>
                    {landers.map((l) => (
                      <option key={l.id} value={l.id}>
                        {l.name}
                      </option>
                    ))}
                  </select>
                </label>
                <label className="form-field" htmlFor="flow-lander-weight">
                  Lander weight
                  <input
                    id="flow-lander-weight"
                    className="form-input form-input--sm"
                    inputMode="numeric"
                    value={flowForm.landerWeight}
                    onChange={(e) => setFlowForm((f) => ({ ...f, landerWeight: e.target.value }))}
                  />
                </label>
              </div>
              <div className="grid-2">
                <label className="form-field" htmlFor="flow-offer">
                  Offer
                  <select
                    id="flow-offer"
                    className="form-input form-input--sm"
                    value={flowForm.offerId}
                    onChange={(e) => setFlowForm((f) => ({ ...f, offerId: e.target.value }))}
                  >
                    <option value="">Select…</option>
                    {offers.map((o) => (
                      <option key={o.id} value={o.id}>
                        {o.name}
                      </option>
                    ))}
                  </select>
                </label>
                <label className="form-field" htmlFor="flow-offer-weight">
                  Offer weight
                  <input
                    id="flow-offer-weight"
                    className="form-input form-input--sm"
                    inputMode="numeric"
                    value={flowForm.offerWeight}
                    onChange={(e) => setFlowForm((f) => ({ ...f, offerWeight: e.target.value }))}
                  />
                </label>
              </div>
              <Button
                label={busy ? 'Saving…' : 'Create flow'}
                variant="primary"
                size="sm"
                loading={busy}
                disabled={busy || landers.length === 0 || offers.length === 0}
                data-testid="flow-create-submit"
                onClick={() => void submitFlow()}
              />
            </section>
          ) : null}
          <section className="section-card">
            <table className="data-table" data-testid="flow-flows-table">
              <thead>
                <tr>
                  <th>Name</th>
                  <th>Paths</th>
                  <th>ID</th>
                </tr>
              </thead>
              <tbody>
                {loading ? <TableSkeleton cols={3} /> : null}
                {!loading && flows.length === 0 ? (
                  <tr>
                    <td colSpan={3} className="text-muted">
                      No flows yet
                    </td>
                  </tr>
                ) : null}
                {!loading
                  ? flows.map((row) => (
                      <tr key={row.id}>
                        <td>{row.name}</td>
                        <td className="text-sm">{summarizeFlowPaths(parseFlowPaths(row.paths))}</td>
                        <td>
                          <CopyableUuid uuid={row.id} />
                        </td>
                      </tr>
                    ))
                  : null}
              </tbody>
            </table>
          </section>
        </div>
      ) : null}
    </>
  );
}
