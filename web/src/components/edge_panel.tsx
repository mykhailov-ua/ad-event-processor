import { displayLabel } from '../helpers/display_labels.js';
import { fraudTierBandRows } from '../helpers/edge_fraud_tier.js';
import { StatusBadge } from './status_badge.js';

export type EdgePanelData = {
  updated_at?: string;
  ingress_h1?: number;
  ingress_h2?: number;
  ingress_h3?: number;
  body_stream?: number;
  body_peek?: number;
  body_read?: number;
  blocked?: Record<string, number>;
  tarpit_total?: number;
  blacklist_stale?: number;
};

export type XDPPanelData = {
  updated_at?: string;
  pass?: number;
  pass_allowlist?: number;
  fingerprints?: number;
  drops?: Record<string, number>;
};

function ingressPct(n: number, total: number): string {
  return total > 0 ? ((n / total) * 100).toFixed(1) : '0.0';
}

/**
 * Edge ingress and block-reason metrics for the operator dashboard.
 */
export function EdgePanel({ edge }: { edge?: EdgePanelData | null }) {
  if (!edge) return null;

  const ingressTotal = (edge.ingress_h1 ?? 0) + (edge.ingress_h2 ?? 0) + (edge.ingress_h3 ?? 0);
  const blocked = edge.blocked ?? {};
  const blockKeys = Object.keys(blocked).sort();

  return (
    <section data-testid="edge-panel" className="settings-panel section-block">
      <div className="settings-panel__header">
        <h2 className="settings-panel__title">Edge traffic</h2>
        {edge.updated_at ? (
          <p className="settings-panel__desc">{`Updated ${edge.updated_at}`}</p>
        ) : null}
      </div>
      <div className="settings-panel__body panel-stack">
        <h3 className="subsection-title">Ingress protocol</h3>
        <dl className="definition-list">
          <dt>HTTP/1.1</dt>
          <dd>{`${edge.ingress_h1 ?? 0} (${ingressPct(edge.ingress_h1 ?? 0, ingressTotal)}%)`}</dd>
          <dt>HTTP/2</dt>
          <dd>{`${edge.ingress_h2 ?? 0} (${ingressPct(edge.ingress_h2 ?? 0, ingressTotal)}%)`}</dd>
          <dt>HTTP/3</dt>
          <dd>{`${edge.ingress_h3 ?? 0} (${ingressPct(edge.ingress_h3 ?? 0, ingressTotal)}%)`}</dd>
        </dl>

        <h3 className="subsection-title">Body handling</h3>
        <dl className="definition-list">
          <dt>Stream</dt>
          <dd>{String(edge.body_stream ?? 0)}</dd>
          <dt>Peek</dt>
          <dd>{String(edge.body_peek ?? 0)}</dd>
          <dt>Read</dt>
          <dd>{String(edge.body_read ?? 0)}</dd>
        </dl>

        <h3 className="subsection-title">Fraud tier (edge)</h3>
        <p className="text-muted text-sm">
          Score bands match edge-fraud-tier.lua. Blocked at edge:{' '}
          <strong className="font-mono">{String(edge.blocked?.fraud_tier ?? 0)}</strong>
        </p>
        <div className="table-wrapper">
          <table className="data-table" data-testid="edge-fraud-tier-table">
            <thead>
              <tr>
                <th scope="col">Tier</th>
                <th scope="col">Score</th>
                <th scope="col">Action</th>
              </tr>
            </thead>
            <tbody>
              {fraudTierBandRows().map((row) => (
                <tr key={row.tier}>
                  <td>
                    <StatusBadge status={row.tier} label={row.tier} />
                  </td>
                  <td className="font-mono">{row.range}</td>
                  <td>{row.action}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        <h3 className="subsection-title">Bot signals</h3>
        <dl className="definition-list">
          <dt>Tarpit delays</dt>
          <dd>{String(edge.tarpit_total ?? 0)}</dd>
          <dt>Blacklist stale rejects</dt>
          <dd>{String(edge.blacklist_stale ?? 0)}</dd>
          <dt>TLS fingerprint (X-TLS-Hash)</dt>
          <dd className="text-muted text-sm">
            Set at edge ingress; blocklist via platform TLS hash config
          </dd>
        </dl>

        <h3 className="subsection-title">Block reasons</h3>
        {blockKeys.length > 0 ? (
          <div className="table-wrapper">
            <table className="data-table">
              <thead>
                <tr>
                  <th scope="col">Reason</th>
                  <th scope="col">Count</th>
                </tr>
              </thead>
              <tbody>
                {blockKeys.map((key) => (
                  <tr key={key}>
                    <td>{displayLabel(key)}</td>
                    <td className="font-mono">{String(blocked[key] ?? 0)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : (
          <p className="text-muted text-sm">No block counters yet.</p>
        )}
      </div>
    </section>
  );
}

/**
 * XDP pass/drop panel for the operator dashboard.
 */
export function XDPPanel({ xdp }: { xdp?: XDPPanelData | null }) {
  if (!xdp || (!xdp.pass && !xdp.fingerprints && !xdp.drops)) return null;

  const drops = xdp.drops ?? {};
  const dropKeys = Object.keys(drops).sort();

  return (
    <section data-testid="xdp-panel" className="settings-panel section-block">
      <div className="settings-panel__header">
        <h2 className="settings-panel__title">Edge packet filter</h2>
      </div>
      <div className="settings-panel__body panel-stack">
        <dl className="definition-list">
          <dt>Pass</dt>
          <dd>{String(xdp.pass ?? 0)}</dd>
          <dt>Allowlist pass</dt>
          <dd>{String(xdp.pass_allowlist ?? 0)}</dd>
          <dt>Fingerprints</dt>
          <dd>{String(xdp.fingerprints ?? 0)}</dd>
        </dl>
        {dropKeys.length > 0 ? (
          <div className="table-wrapper">
            <table className="data-table">
              <thead>
                <tr>
                  <th scope="col">Reason</th>
                  <th scope="col">Drops</th>
                </tr>
              </thead>
              <tbody>
                {dropKeys.map((key) => (
                  <tr key={key}>
                    <td>{displayLabel(key)}</td>
                    <td className="font-mono">{String(drops[key] ?? 0)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : null}
      </div>
    </section>
  );
}
