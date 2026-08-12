import { el } from '../lib/dom.js';
import { renderStatusBadge } from './status_badge.js';
import { displayLabel } from '../helpers/display_labels.js';
import { humanizeTechnicalDetail } from '../helpers/technical_labels.js';

/**
 * Split items into columns with ceil(n/cols) rows each.
 */
function splitColumns<T>(items: T[], cols = 2): T[][] {
  if (items.length === 0) return [];
  const perCol = Math.ceil(items.length / cols);
  const columns: T[][] = [];
  for (let c = 0; c < cols; c += 1) {
    const slice = items.slice(c * perCol, (c + 1) * perCol);
    if (slice.length > 0) columns.push(slice);
  }
  return columns;
}

/**
 * Render a labeled doctor stack section with two-column layout.
 */
function renderStackSection(label: string, rows: HTMLElement[]): HTMLElement | null {
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

export type DoctorStackRow = {
  title: string;
  detail?: string | null;
  mono?: boolean;
  status: string | undefined;
};

/**
 * Render one doctor stack row with title, detail, and status badge.
 */
function renderStackRow(row: DoctorStackRow): HTMLElement {
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
      renderStatusBadge(row.status ?? '', { kind: 'service' }),
    ),
  );
}

/** Pinned doctor probes shown in a dedicated platform section. */
const PINNED_CHECK_IDS = ['license', 'slotmap'] as const;

export type DoctorCheck = {
  id?: string;
  status?: string;
  message?: string;
  hint?: string;
};

/**
 * Render pinned platform probes (license, slot map) with remediation hints.
 */
function renderPinnedChecks(checks: DoctorCheck[]): HTMLElement | null {
  const rows: HTMLElement[] = [];
  for (let i = 0; i < PINNED_CHECK_IDS.length; i++) {
    const want = PINNED_CHECK_IDS[i];
    const check = checks.find((c) => c.id === want);
    if (!check) continue;
    rows.push(el('div', { className: 'doctor-pinned-check' },
      renderStackRow({
        title: displayLabel(check.id),
        detail: humanizeTechnicalDetail(check.message),
        status: check.status,
      }),
      check.hint
        ? el('p', { className: 'doctor-stack__hint text-muted text-sm' }, check.hint)
        : null,
    ));
  }
  if (rows.length === 0) return null;
  return renderStackSection('Platform', rows);
}

export type DoctorService = {
  name?: string;
  status?: string;
  detail?: string;
};

export type DoctorPanelOpts = {
  doctor?: { overall?: string; checks?: DoctorCheck[] } | null;
  services?: DoctorService[] | null;
  loading?: boolean;
};

/**
 * Render the operations doctor panel with services and checks.
 */
export function renderDoctorPanel(opts: DoctorPanelOpts): HTMLElement {
  const services = opts.services ?? [];
  const allChecks = opts.doctor?.checks ?? [];
  const pinnedIds = new Set<string>(PINNED_CHECK_IDS);
  const checks = allChecks.filter((c) => !pinnedIds.has(c.id ?? ''));
  const loading = opts.loading ?? false;

  const pinnedStack = renderPinnedChecks(allChecks);

  const serviceRows = services.map((svc) =>
    renderStackRow({
      title: displayLabel(svc.name),
      detail: humanizeTechnicalDetail(svc.detail),
      status: svc.status,
    }),
  );

  const checkRows = checks.map((check) =>
    renderStackRow({
      title: displayLabel(check.id),
      detail: humanizeTechnicalDetail(check.message),
      status: check.status,
    }),
  );

  const serviceStack = renderStackSection('', serviceRows);
  const checkStack = renderStackSection('Checks', checkRows);

  return el('section', { className: 'doctor-panel' },
    el('div', { className: 'doctor-panel__header' },
      el('h2', { className: 'doctor-panel__title' }, 'Doctor'),
      loading
        ? el('span', { className: 'text-muted text-sm' }, 'Loading…')
        : opts.doctor?.overall
          ? renderStatusBadge(opts.doctor.overall, { kind: 'service' })
          : null,
    ),
    pinnedStack,
    serviceStack,
    checkStack,
    !loading && services.length === 0 && allChecks.length === 0
      ? el('p', { className: 'doctor-panel__empty text-muted' }, 'No health data yet.')
      : null,
  );
}
