import { el } from '../lib/dom.js';
import { displayLabel } from '../helpers/display_labels.js';
import { fraudTierBandRows } from '../helpers/edge_fraud_tier.js';
import { renderStatusBadge } from './status_badge.js';

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

/**
 * Render edge ingress and block-reason metrics panel.
 */
export function renderEdgePanel(edge: EdgePanelData | null | undefined): HTMLElement | null {
  if (!edge) return null;
  const ingressTotal = (edge.ingress_h1 ?? 0) + (edge.ingress_h2 ?? 0) + (edge.ingress_h3 ?? 0);
  const pct = (n: number): string => (ingressTotal > 0 ? ((n / ingressTotal) * 100).toFixed(1) : '0.0');

  const blocked = edge.blocked ?? {};
  const blockRows = Object.keys(blocked).sort().map((key) =>
    el('tr', null,
      el('td', null, displayLabel(key)),
      el('td', { className: 'font-mono' }, String(blocked[key] ?? 0)),
    ),
  );

  return el('section', { 'data-testid': 'edge-panel', className: 'settings-panel section-block' },
    el('div', { className: 'settings-panel__header' },
      el('h2', { className: 'settings-panel__title' }, 'Edge traffic'),
      edge.updated_at
        ? el('p', { className: 'settings-panel__desc' }, `Updated ${edge.updated_at}`)
        : null,
    ),
    el('div', { className: 'settings-panel__body panel-stack' },
      el('h3', { className: 'subsection-title' }, 'Ingress protocol'),
      el('dl', { className: 'definition-list' },
        el('dt', null, 'HTTP/1.1'),
        el('dd', null, `${edge.ingress_h1 ?? 0} (${pct(edge.ingress_h1 ?? 0)}%)`),
        el('dt', null, 'HTTP/2'),
        el('dd', null, `${edge.ingress_h2 ?? 0} (${pct(edge.ingress_h2 ?? 0)}%)`),
        el('dt', null, 'HTTP/3'),
        el('dd', null, `${edge.ingress_h3 ?? 0} (${pct(edge.ingress_h3 ?? 0)}%)`),
      ),
      el('h3', { className: 'subsection-title' }, 'Body handling'),
      el('dl', { className: 'definition-list' },
        el('dt', null, 'Stream'),
        el('dd', null, String(edge.body_stream ?? 0)),
        el('dt', null, 'Peek'),
        el('dd', null, String(edge.body_peek ?? 0)),
        el('dt', null, 'Read'),
        el('dd', null, String(edge.body_read ?? 0)),
      ),
      el('h3', { className: 'subsection-title' }, 'Fraud tier (edge)'),
      el('p', { className: 'text-muted text-sm' },
        'Score bands match edge-fraud-tier.lua. Blocked at edge: ',
        el('strong', { className: 'font-mono' }, String(edge.blocked?.fraud_tier ?? 0)),
      ),
      el('div', { className: 'table-wrapper' },
        el('table', { className: 'data-table', 'data-testid': 'edge-fraud-tier-table' },
          el('thead', null,
            el('tr', null,
              el('th', { scope: 'col' }, 'Tier'),
              el('th', { scope: 'col' }, 'Score'),
              el('th', { scope: 'col' }, 'Action'),
            ),
          ),
          el('tbody', null,
            fraudTierBandRows().map((row) => el('tr', null,
              el('td', null, renderStatusBadge(row.tier, { label: row.tier })),
              el('td', { className: 'font-mono' }, row.range),
              el('td', null, row.action),
            )),
          ),
        ),
      ),
      el('h3', { className: 'subsection-title' }, 'Bot signals'),
      el('dl', { className: 'definition-list' },
        el('dt', null, 'Tarpit delays'),
        el('dd', null, String(edge.tarpit_total ?? 0)),
        el('dt', null, 'Blacklist stale rejects'),
        el('dd', null, String(edge.blacklist_stale ?? 0)),
        el('dt', null, 'TLS fingerprint (X-TLS-Hash)'),
        el('dd', { className: 'text-muted text-sm' }, 'Set at edge ingress; blocklist via platform TLS hash config'),
      ),
      el('h3', { className: 'subsection-title' }, 'Block reasons'),
      blockRows.length > 0
        ? el('div', { className: 'table-wrapper' },
          el('table', { className: 'data-table' },
            el('thead', null,
              el('tr', null,
                el('th', { scope: 'col' }, 'Reason'),
                el('th', { scope: 'col' }, 'Count'),
              ),
            ),
            el('tbody', null, ...blockRows),
          ),
        )
        : el('p', { className: 'text-muted text-sm' }, 'No block counters yet.'),
    ),
  );
}

export type XDPPanelData = {
  updated_at?: string;
  pass?: number;
  pass_allowlist?: number;
  fingerprints?: number;
  drops?: Record<string, number>;
};

/**
 * Render XDP pass/drop panel for operator dashboard.
 */
export function renderXDPPanel(xdp: XDPPanelData | null | undefined): HTMLElement | null {
  if (!xdp || (!xdp.pass && !xdp.fingerprints && !xdp.drops)) return null;
  const drops = xdp.drops ?? {};
  const dropRows = Object.keys(drops).sort().map((key) =>
    el('tr', null,
      el('td', null, displayLabel(key)),
      el('td', { className: 'font-mono' }, String(drops[key] ?? 0)),
    ),
  );
  return el('section', { 'data-testid': 'xdp-panel', className: 'settings-panel section-block' },
    el('div', { className: 'settings-panel__header' },
      el('h2', { className: 'settings-panel__title' }, 'Edge packet filter'),
    ),
    el('div', { className: 'settings-panel__body panel-stack' },
      el('dl', { className: 'definition-list' },
        el('dt', null, 'Pass'),
        el('dd', null, String(xdp.pass ?? 0)),
        el('dt', null, 'Allowlist pass'),
        el('dd', null, String(xdp.pass_allowlist ?? 0)),
        el('dt', null, 'Fingerprints'),
        el('dd', null, String(xdp.fingerprints ?? 0)),
      ),
      dropRows.length > 0
        ? el('div', { className: 'table-wrapper' },
          el('table', { className: 'data-table' },
            el('thead', null,
              el('tr', null,
                el('th', { scope: 'col' }, 'Reason'),
                el('th', { scope: 'col' }, 'Drops'),
              ),
            ),
            el('tbody', null, ...dropRows),
          ),
        )
        : null,
    ),
  );
}
