import { useCallback, useEffect, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { to } from '../lib/to.js';
import * as auth from '../helpers/auth.js';
import { can } from '../helpers/permissions.js';
import { mapServiceError } from '../helpers/service_error.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import {
  appendLanderRow,
  appendOfferRow,
  emptyFlowPath,
  moveFlowBuilderItem,
  totalFlowPathWeight,
} from '../helpers/flow_builder_model.js';
import {
  fetchFlow,
  fetchLanders,
  fetchOffers,
  parseFlowPaths,
  updateFlow,
  type FlowPathDTO,
  type LanderDTO,
  type OfferDTO,
} from '../helpers/flows_api.js';
import { Breadcrumbs } from '../components/breadcrumbs.js';
import { Button } from '../components/button.js';
import { ErrorBlock } from '../components/error_block.js';

/**
 * Parse a positive integer from builder weight inputs.
 * @param raw - Input string from a weight field.
 */
function parsePositiveWeight(raw: string): number | null {
  const value = Number.parseInt(raw, 10);
  if (!Number.isFinite(value) || value <= 0) return null;
  return value;
}

export function FlowBuilderPage() {
  const { id = '' } = useParams();
  const canWrite = can(auth.getUser()?.permissions ?? [], 'campaigns:write');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<unknown>(null);
  const [busy, setBusy] = useState(false);
  const [flowName, setFlowName] = useState('');
  const [paths, setPaths] = useState<FlowPathDTO[]>([]);
  const [landers, setLanders] = useState<LanderDTO[]>([]);
  const [offers, setOffers] = useState<OfferDTO[]>([]);

  const reload = useCallback(async () => {
    if (!id) {
      setError(new Error('Missing flow id'));
      setLoading(false);
      return;
    }
    setLoading(true);
    setError(null);
    const [flowRes, landerRes, offerRes] = await Promise.all([
      to(fetchFlow(id)),
      to(fetchLanders()),
      to(fetchOffers()),
    ]);
    setLoading(false);
    if (flowRes[1]) {
      setError(flowRes[1]);
      return;
    }
    const flow = flowRes[0];
    if (!flow) {
      setError(new Error('Flow not found'));
      return;
    }
    setFlowName(flow.name);
    const parsed = parseFlowPaths(flow.paths);
    setPaths(parsed.length ? parsed : [emptyFlowPath()]);
    setLanders(landerRes[1] ? [] : (landerRes[0] ?? []));
    setOffers(offerRes[1] ? [] : (offerRes[0] ?? []));
  }, [id]);

  useEffect(() => {
    void reload();
  }, [reload]);

  const onSave = async () => {
    if (!canWrite || busy || !id) return;
    const name = flowName.trim();
    if (!name) {
      pushToastMessage({ title: 'Missing name', message: 'Flow name is required' });
      return;
    }
    if (!paths.length) {
      pushToastMessage({ title: 'Missing paths', message: 'Add at least one path' });
      return;
    }
    setBusy(true);
    const [, err] = await to(updateFlow(id, name, paths));
    setBusy(false);
    if (err) {
      if (err instanceof ConfirmCancelledError) return;
      pushToastMessage({ title: 'Save failed', message: mapServiceError(err).message });
      return;
    }
    pushToastMessage({ title: 'Flow saved', message: name });
    await reload();
  };

  const updatePath = (index: number, next: FlowPathDTO) => {
    setPaths((rows) => rows.map((row, i) => (i === index ? next : row)));
  };

  if (error) {
    return <ErrorBlock error={error} />;
  }

  const defaultLanderId = landers[0]?.id ?? '';
  const defaultOfferId = offers[0]?.id ?? '';

  return (
    <>
      <div className="page-header">
        <Breadcrumbs
          items={[
            { label: 'Campaigns', href: '/campaigns' },
            { label: 'Flows', href: '/campaigns/flows' },
            { label: flowName || 'Flow builder' },
          ]}
        />
        <h1 className="page-header__title">Flow builder</h1>
        <p className="text-muted text-sm">
          Weighted paths for /click routing. Each path splits traffic between landers and offers.
        </p>
      </div>

      <div className="section-block stack">
        <div className="toolbar-row">
          {canWrite ? (
            <Button
              label={busy ? 'Saving...' : 'Save flow'}
              variant="primary"
              size="sm"
              loading={busy}
              disabled={busy || loading}
              data-testid="flow-builder-save"
              onClick={() => void onSave()}
            />
          ) : null}
          {canWrite ? (
            <Button
              label="Add path"
              variant="secondary"
              size="sm"
              disabled={busy || loading}
              data-testid="flow-builder-add-path"
              onClick={() =>
                setPaths((rows) => [...rows, emptyFlowPath(defaultLanderId, defaultOfferId)])
              }
            />
          ) : null}
          <Link className="button button--sm button--ghost" to="/campaigns/flows">
            Back to flows
          </Link>
        </div>

        <section className="section-card stack" data-testid="flow-builder-meta">
          <label className="form-field" htmlFor="flow-builder-name">
            Flow name
            <input
              id="flow-builder-name"
              className="form-input form-input--sm"
              value={flowName}
              disabled={!canWrite || loading}
              onChange={(e) => setFlowName(e.target.value)}
            />
          </label>
          <p className="text-sm text-muted">
            Path weight total: {totalFlowPathWeight(paths)} (split across {paths.length} path
            {paths.length === 1 ? '' : 's'})
          </p>
        </section>

        {loading ? <p className="text-muted text-sm">Loading...</p> : null}

        {!loading
          ? paths.map((path, pathIndex) => (
              <section
                key={`path-${pathIndex}`}
                className="section-card stack"
                data-testid={`flow-builder-path-${pathIndex}`}
              >
                <div className="toolbar-row">
                  <h3 className="subsection-title">Path {pathIndex + 1}</h3>
                  {canWrite ? (
                    <>
                      <Button
                        label="Up"
                        variant="ghost"
                        size="sm"
                        disabled={pathIndex === 0}
                        onClick={() =>
                          setPaths((rows) => moveFlowBuilderItem(rows, pathIndex, -1))
                        }
                      />
                      <Button
                        label="Down"
                        variant="ghost"
                        size="sm"
                        disabled={pathIndex >= paths.length - 1}
                        onClick={() => setPaths((rows) => moveFlowBuilderItem(rows, pathIndex, 1))}
                      />
                      <Button
                        label="Remove"
                        variant="ghost"
                        size="sm"
                        disabled={paths.length <= 1}
                        onClick={() => setPaths((rows) => rows.filter((_, i) => i !== pathIndex))}
                      />
                    </>
                  ) : null}
                </div>

                <label className="form-field" htmlFor={`path-weight-${pathIndex}`}>
                  Path weight
                  <input
                    id={`path-weight-${pathIndex}`}
                    className="form-input form-input--sm"
                    inputMode="numeric"
                    value={String(path.weight)}
                    disabled={!canWrite}
                    onChange={(e) => {
                      const weight = parsePositiveWeight(e.target.value);
                      if (weight == null) return;
                      updatePath(pathIndex, { ...path, weight });
                    }}
                  />
                </label>

                <div className="grid-2">
                  <div className="stack">
                    <h4 className="text-sm">Landers</h4>
                    {path.landers.map((lander, landerIndex) => (
                      <div key={`l-${pathIndex}-${landerIndex}`} className="grid-2">
                        <label className="form-field">
                          Lander
                          <select
                            className="form-input form-input--sm"
                            value={lander.lander_id}
                            disabled={!canWrite}
                            onChange={(e) => {
                              const nextLanders = path.landers.map((row, i) =>
                                i === landerIndex ? { ...row, lander_id: e.target.value } : row
                              );
                              updatePath(pathIndex, { ...path, landers: nextLanders });
                            }}
                          >
                            <option value="">Select...</option>
                            {landers.map((row) => (
                              <option key={row.id} value={row.id}>
                                {row.name}
                              </option>
                            ))}
                          </select>
                        </label>
                        <label className="form-field">
                          Weight
                          <input
                            className="form-input form-input--sm"
                            inputMode="numeric"
                            value={String(lander.weight)}
                            disabled={!canWrite}
                            onChange={(e) => {
                              const weight = parsePositiveWeight(e.target.value);
                              if (weight == null) return;
                              const nextLanders = path.landers.map((row, i) =>
                                i === landerIndex ? { ...row, weight } : row
                              );
                              updatePath(pathIndex, { ...path, landers: nextLanders });
                            }}
                          />
                        </label>
                        {canWrite ? (
                          <Button
                            label="Remove"
                            variant="ghost"
                            size="sm"
                            disabled={path.landers.length <= 1}
                            onClick={() =>
                              updatePath(pathIndex, {
                                ...path,
                                landers: path.landers.filter((_, i) => i !== landerIndex),
                              })
                            }
                          />
                        ) : null}
                      </div>
                    ))}
                    {canWrite ? (
                      <Button
                        label="Add lander"
                        variant="secondary"
                        size="sm"
                        onClick={() => updatePath(pathIndex, appendLanderRow(path, defaultLanderId))}
                      />
                    ) : null}
                  </div>

                  <div className="stack">
                    <h4 className="text-sm">Offers</h4>
                    {path.offers.map((offer, offerIndex) => (
                      <div key={`o-${pathIndex}-${offerIndex}`} className="grid-2">
                        <label className="form-field">
                          Offer
                          <select
                            className="form-input form-input--sm"
                            value={offer.offer_id}
                            disabled={!canWrite}
                            onChange={(e) => {
                              const nextOffers = path.offers.map((row, i) =>
                                i === offerIndex ? { ...row, offer_id: e.target.value } : row
                              );
                              updatePath(pathIndex, { ...path, offers: nextOffers });
                            }}
                          >
                            <option value="">Select...</option>
                            {offers.map((row) => (
                              <option key={row.id} value={row.id}>
                                {row.name}
                              </option>
                            ))}
                          </select>
                        </label>
                        <label className="form-field">
                          Weight
                          <input
                            className="form-input form-input--sm"
                            inputMode="numeric"
                            value={String(offer.weight)}
                            disabled={!canWrite}
                            onChange={(e) => {
                              const weight = parsePositiveWeight(e.target.value);
                              if (weight == null) return;
                              const nextOffers = path.offers.map((row, i) =>
                                i === offerIndex ? { ...row, weight } : row
                              );
                              updatePath(pathIndex, { ...path, offers: nextOffers });
                            }}
                          />
                        </label>
                        {canWrite ? (
                          <Button
                            label="Remove"
                            variant="ghost"
                            size="sm"
                            disabled={path.offers.length <= 1}
                            onClick={() =>
                              updatePath(pathIndex, {
                                ...path,
                                offers: path.offers.filter((_, i) => i !== offerIndex),
                              })
                            }
                          />
                        ) : null}
                      </div>
                    ))}
                    {canWrite ? (
                      <Button
                        label="Add offer"
                        variant="secondary"
                        size="sm"
                        onClick={() => updatePath(pathIndex, appendOfferRow(path, defaultOfferId))}
                      />
                    ) : null}
                  </div>
                </div>
              </section>
            ))
          : null}
      </div>
    </>
  );
}
