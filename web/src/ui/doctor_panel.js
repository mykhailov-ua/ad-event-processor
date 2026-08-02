import { el } from '../lib/dom.js';
import { renderStatusBadge } from './status_badge.js';

/**
 * Split into columns with ceil(n/cols) rows each (table-style).
 * @param {unknown[]} items
 * @param {number} cols
 * @returns {unknown[][]}
 */
function splitColumns(items, cols = 2) {
  if (items.length === 0) return [];
  const perCol = Math.ceil(items.length / cols);
  const columns = [];
  for (let c = 0; c < cols; c += 1) {
    const slice = items.slice(c * perCol, (c + 1) * perCol);
    if (slice.length > 0) columns.push(slice);
  }
  return columns;
}

/**
 * @param {string} label
 * @param {HTMLElement[]} rows
 */
function renderStackSection(label, rows) {
  if (rows.length === 0) return null;
  const columns = splitColumns(rows, 2);
  return el('div', { className: 'doctor-stack-section' },
    label
      ? el('div', { className: 'doctor-stack-section__label' }, label)
      : null,
    el('div', { className: 'doctor-stack', role: 'table' },
      columns.map((colRows) =>
        el('div', { className: 'doctor-stack__col', role: 'rowgroup' },
          colRows,
        ),
      ),
    ),
  );
}

/**
 * @param {{
 *   title: string,
 *   detail?: string | null,
 *   mono?: boolean,
 *   status: string | undefined,
 * }} row
 */
function renderStackRow(row) {
  return el('div', { className: 'doctor-stack__row', role: 'row' },
    el('div', { className: 'doctor-stack__body' },
      el('div', {
        className: [
          'doctor-stack__title',
          row.mono ? 'font-mono' : '',
        ].filter(Boolean).join(' '),
      }, row.title),
      row.detail
        ? el('div', { className: 'doctor-stack__detail text-muted' }, row.detail)
        : null,
    ),
    el('div', { className: 'doctor-stack__badge' },
      renderStatusBadge(row.status, { kind: 'service' }),
    ),
  );
}

/**
 * @param {{
 *   doctor?: { overall?: string, checks?: Array<{ id?: string, status?: string, message?: string }> } | null,
 *   services?: Array<{ name?: string, status?: string, detail?: string }> | null,
 *   loading?: boolean,
 * }} opts
 */
export function renderDoctorPanel(opts) {
  const services = opts.services ?? [];
  const checks = opts.doctor?.checks ?? [];
  const loading = opts.loading ?? false;

  const serviceRows = services.map((svc) =>
    renderStackRow({
      title: svc.name ?? '—',
      detail: svc.detail,
      status: svc.status,
    }),
  );

  const checkRows = checks.map((check) =>
    renderStackRow({
      title: check.id ?? '—',
      detail: check.message,
      mono: true,
      status: check.status,
    }),
  );

  const serviceStack = renderStackSection('', serviceRows);
  const checkStack = renderStackSection('Checks', checkRows);

  return el('section', { className: 'doctor-panel' },
    el('div', { className: 'doctor-panel__header' },
      el('h2', { className: 'doctor-panel__title' }, 'Doctor'),
      loading
        ? el('span', { className: 'text-muted', style: { fontSize: 13 } }, 'Loading…')
        : opts.doctor?.overall
          ? renderStatusBadge(opts.doctor.overall, { kind: 'service', label: opts.doctor.overall })
          : null,
    ),
    serviceStack,
    checkStack,
    !loading && services.length === 0 && checks.length === 0
      ? el('p', { className: 'doctor-panel__empty text-muted' }, 'No health data yet.')
      : null,
  );
}
