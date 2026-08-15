import { useCallback, useEffect, useState } from 'react';
import { to } from '../../lib/to.js';
import * as auth from '../../helpers/auth.js';
import { can } from '../../helpers/permissions.js';
import { mapServiceError } from '../../helpers/service_error.js';
import { pushToastMessage } from '../../helpers/toast_ui.js';
import { ConfirmCancelledError } from '../../helpers/confirm_ui.js';
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
} from '../../helpers/supply_api.js';
import { Button } from '../components/button.js';
import { ErrorBlock } from '../components/error_block.js';

type SellerRow = {
  id: number;
  seller_id: string;
  domain: string;
  seller_type: string;
  name?: string;
};

type AdsTxtRow = {
  id: number;
  domain: string;
  publisher_account_id: string;
  relationship: string;
};

type SupplyTab = 'sellers' | 'ads' | 'preview';

function TableSkeleton({ cols, rows = 4 }: { cols: number; rows?: number }) {
  return (
    <>
      {Array.from({ length: rows }, (_, i) => (
        <tr key={`sk-${i}`} className="data-table__row--skeleton" aria-hidden="true">
          {Array.from({ length: cols }, (__, j) => (
            <td key={`sk-${i}-${j}`}><span className="skeleton-bar" /></td>
          ))}
        </tr>
      ))}
    </>
  );
}

/**
 * Supply files admin (sellers.json + ads.txt).
 */
export function IntegrationsSupplyPage() {
  const canWrite = can(auth.getUser()?.permissions ?? [], 'settings:write');

  const [tab, setTab] = useState<SupplyTab>('sellers');
  const [sellers, setSellers] = useState<SellerRow[]>([]);
  const [adsRows, setAdsRows] = useState<AdsTxtRow[]>([]);
  const [exportPath, setExportPath] = useState('');
  const [sellersPreview, setSellersPreview] = useState('');
  const [adsPreview, setAdsPreview] = useState('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<unknown>(null);
  const [busy, setBusy] = useState(false);

  const [sellerForm, setSellerForm] = useState({
    seller_id: '',
    domain: '',
    seller_type: 'PUBLISHER',
    name: '',
    is_confidential: false,
  });
  const [adsForm, setAdsForm] = useState({
    domain: '',
    publisher_account_id: '',
    relationship: 'DIRECT',
    cert_authority_id: '',
    sort_order: '0',
  });

  const reload = useCallback(async () => {
    setLoading(true);
    setError(null);
    const [sRes, aRes, pRes] = await Promise.all([
      to(fetchSellers()),
      to(fetchAdsTxtEntries()),
      to(fetchSupplyExportPath()),
    ]);
    setLoading(false);
    if (sRes[1]) {
      setError(sRes[1]);
      return;
    }
    setSellers((sRes[0] ?? []) as SellerRow[]);
    setAdsRows(aRes[1] ? [] : ((aRes[0] ?? []) as AdsTxtRow[]));
    setExportPath(pRes[1] ? '' : (pRes[0] ?? ''));
  }, []);

  const loadPreviews = useCallback(async () => {
    const [s, a] = await Promise.all([
      to(fetchSellersJSONPreview()),
      to(fetchAdsTxtPreview()),
    ]);
    setSellersPreview(s[1] ? `Error: ${s[1].message}` : (s[0] ?? ''));
    setAdsPreview(a[1] ? `Error: ${a[1].message}` : (a[0] ?? ''));
  }, []);

  useEffect(() => {
    void reload();
  }, [reload]);

  useEffect(() => {
    if (tab === 'preview') void loadPreviews();
  }, [tab, loadPreviews]);

  const addSeller = async () => {
    if (!canWrite) return;
    setBusy(true);
    const [, err] = await to(createSeller(sellerForm));
    setBusy(false);
    if (err) {
      if (err instanceof ConfirmCancelledError) return;
      pushToastMessage({ title: 'Seller create failed', message: mapServiceError(err).message });
      return;
    }
    setSellerForm((f) => ({ ...f, seller_id: '', domain: '', name: '' }));
    void reload();
  };

  const addAdsRow = async () => {
    if (!canWrite) return;
    setBusy(true);
    const [, err] = await to(createAdsTxtEntry({
      domain: adsForm.domain.trim(),
      publisher_account_id: adsForm.publisher_account_id.trim(),
      relationship: adsForm.relationship.trim(),
      cert_authority_id: adsForm.cert_authority_id.trim(),
      sort_order: Number.parseInt(adsForm.sort_order, 10) || 0,
    }));
    setBusy(false);
    if (err) {
      if (err instanceof ConfirmCancelledError) return;
      pushToastMessage({ title: 'ads.txt row failed', message: mapServiceError(err).message });
      return;
    }
    setAdsForm((f) => ({ ...f, domain: '', publisher_account_id: '' }));
    void reload();
  };

  if (error) {
    return <ErrorBlock error={error} fallbackTitle="Supply admin unavailable" />;
  }

  return (
    <section className="stack" data-testid="supply-admin-view">
      <div className="page-header">
        <h1 className="page-header__title">Supply files</h1>
        <p className="page-header__desc">
          Manage sellers.json and ads.txt. Export path:{' '}
          <code className="code-inline">{exportPath || '—'}</code>
        </p>
      </div>

      <div className="filter-row cluster--actions">
        <Button
          label="Sellers"
          variant={tab === 'sellers' ? 'primary' : 'secondary'}
          size="sm"
          onClick={() => setTab('sellers')}
        />
        <Button
          label="ads.txt"
          variant={tab === 'ads' ? 'primary' : 'secondary'}
          size="sm"
          onClick={() => setTab('ads')}
        />
        <Button
          label="Preview"
          variant={tab === 'preview' ? 'primary' : 'secondary'}
          size="sm"
          onClick={() => setTab('preview')}
        />
      </div>

      {tab === 'sellers' ? (
        <div className="section-card stack">
          <div className="table-wrapper">
            <table className="data-table">
              <thead>
                <tr>
                  <th>Seller ID</th>
                  <th>Domain</th>
                  <th>Type</th>
                  <th>Name</th>
                  <th />
                </tr>
              </thead>
              <tbody>
                {loading ? <TableSkeleton cols={5} /> : null}
                {sellers.map((s) => (
                  <tr key={s.id}>
                    <td className="font-mono">{s.seller_id}</td>
                    <td>{s.domain}</td>
                    <td>{s.seller_type}</td>
                    <td>{s.name || '—'}</td>
                    <td>
                      {canWrite ? (
                        <Button
                          label="Delete"
                          variant="secondary"
                          size="sm"
                          onClick={async () => {
                            const [, err] = await to(deleteSeller(s.id));
                            if (!err) void reload();
                          }}
                        />
                      ) : null}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {canWrite ? (
            <div className="stack mt-4">
              <h3 className="subsection-title">Add seller</h3>
              <div className="form-row">
                <label className="form-field">
                  Seller ID
                  <input
                    className="form-input"
                    value={sellerForm.seller_id}
                    onChange={(e) => setSellerForm((f) => ({ ...f, seller_id: e.target.value }))}
                  />
                </label>
                <label className="form-field">
                  Domain
                  <input
                    className="form-input"
                    value={sellerForm.domain}
                    onChange={(e) => setSellerForm((f) => ({ ...f, domain: e.target.value }))}
                  />
                </label>
              </div>
              <div className="form-row">
                <label className="form-field">
                  Type
                  <select
                    className="form-select"
                    value={sellerForm.seller_type}
                    onChange={(e) => setSellerForm((f) => ({ ...f, seller_type: e.target.value }))}
                  >
                    <option value="PUBLISHER">PUBLISHER</option>
                    <option value="INTERMEDIARY">INTERMEDIARY</option>
                    <option value="BOTH">BOTH</option>
                  </select>
                </label>
                <label className="form-field">
                  Name
                  <input
                    className="form-input"
                    value={sellerForm.name}
                    onChange={(e) => setSellerForm((f) => ({ ...f, name: e.target.value }))}
                  />
                </label>
              </div>
              <Button
                label="Add seller"
                variant="primary"
                size="sm"
                loading={busy}
                disabled={busy}
                onClick={() => void addSeller()}
              />
            </div>
          ) : null}
        </div>
      ) : null}

      {tab === 'ads' ? (
        <div className="section-card stack">
          <div className="table-wrapper">
            <table className="data-table">
              <thead>
                <tr>
                  <th>Domain</th>
                  <th>Account ID</th>
                  <th>Relationship</th>
                  <th />
                </tr>
              </thead>
              <tbody>
                {loading ? <TableSkeleton cols={4} /> : null}
                {adsRows.map((r) => (
                  <tr key={r.id}>
                    <td>{r.domain}</td>
                    <td className="font-mono">{r.publisher_account_id}</td>
                    <td>{r.relationship}</td>
                    <td>
                      {canWrite ? (
                        <Button
                          label="Delete"
                          variant="secondary"
                          size="sm"
                          onClick={async () => {
                            const [, err] = await to(deleteAdsTxtEntry(r.id));
                            if (!err) void reload();
                          }}
                        />
                      ) : null}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {canWrite ? (
            <div className="stack mt-4">
              <h3 className="subsection-title">Add ads.txt row</h3>
              <div className="form-row">
                <label className="form-field">
                  Domain
                  <input
                    className="form-input"
                    value={adsForm.domain}
                    onChange={(e) => setAdsForm((f) => ({ ...f, domain: e.target.value }))}
                  />
                </label>
                <label className="form-field">
                  Publisher account ID
                  <input
                    className="form-input"
                    value={adsForm.publisher_account_id}
                    onChange={(e) => setAdsForm((f) => ({ ...f, publisher_account_id: e.target.value }))}
                  />
                </label>
                <label className="form-field">
                  Relationship
                  <select
                    className="form-select"
                    value={adsForm.relationship}
                    onChange={(e) => setAdsForm((f) => ({ ...f, relationship: e.target.value }))}
                  >
                    <option value="DIRECT">DIRECT</option>
                    <option value="RESELLER">RESELLER</option>
                  </select>
                </label>
              </div>
              <Button
                label="Add row"
                variant="primary"
                size="sm"
                loading={busy}
                disabled={busy}
                onClick={() => void addAdsRow()}
              />
            </div>
          ) : null}
        </div>
      ) : null}

      {tab === 'preview' ? (
        <div className="section-card stack">
          <h3 className="subsection-title">sellers.json</h3>
          <pre className="code-block text-sm">{sellersPreview || 'Loading…'}</pre>
          <h3 className="subsection-title">ads.txt</h3>
          <pre className="code-block text-sm">{adsPreview || 'Loading…'}</pre>
        </div>
      ) : null}
    </section>
  );
}
