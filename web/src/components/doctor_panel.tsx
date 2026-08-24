import type { OpsDoctorSummary } from '../types/ops.js';
import { displayLabel } from '../helpers/display_labels.js';
import { humanizeTechnicalDetail } from '../helpers/technical_labels.js';
import { StatusBadge } from './status_badge.js';

const PINNED_CHECK_IDS = ['license', 'slotmap', 'edge_xdp'] as const;

export type DoctorService = {
  name?: string;
  status?: string;
  detail?: string;
};

export type DoctorPanelProps = {
  doctor?: OpsDoctorSummary | null;
  services?: DoctorService[] | null;
  loading?: boolean;
};

type StackRow = {
  title: string;
  detail?: string | null;
  mono?: boolean;
  status: string | undefined;
};

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

function StackRowView({ row }: { row: StackRow }) {
  return (
    <div className="doctor-stack__row" role="row">
      <div className="doctor-stack__body">
        <div className={`doctor-stack__title${row.mono ? ' font-mono' : ''}`}>{row.title}</div>
        {row.detail ? <div className="doctor-stack__detail text-muted">{row.detail}</div> : null}
      </div>
      <div className="doctor-stack__badge">
        <StatusBadge status={row.status ?? ''} kind="service" />
      </div>
    </div>
  );
}

function StackSection({ label, rows }: { label: string; rows: StackRow[] }) {
  if (rows.length === 0) return null;
  const columns = splitColumns(rows, 2);
  return (
    <div className="doctor-stack-section">
      {label ? <div className="doctor-stack-section__label">{label}</div> : null}
      <div className="doctor-stack" role="table">
        {columns.map((colRows, colIndex) => (
          <div key={`col-${colIndex}`} className="doctor-stack__col" role="rowgroup">
            {colRows.map((row, rowIndex) => (
              <StackRowView key={`${row.title}-${rowIndex}`} row={row} />
            ))}
          </div>
        ))}
      </div>
    </div>
  );
}

function PinnedChecks({ checks }: { checks: NonNullable<OpsDoctorSummary['checks']> }) {
  const rows = PINNED_CHECK_IDS.flatMap((want) => {
    const check = checks.find((c) => c.id === want);
    if (!check) return [];
    return [
      {
        check,
        row: {
          title: displayLabel(check.id),
          detail: humanizeTechnicalDetail(check.message),
          status: check.status,
        } satisfies StackRow,
      },
    ];
  });

  if (rows.length === 0) return null;

  return (
    <div className="doctor-stack-section">
      <div className="doctor-stack-section__label">Platform</div>
      <div className="doctor-stack" role="table">
        {rows.map(({ check, row }) => (
          <div key={check.id} className="doctor-pinned-check">
            <StackRowView row={row} />
            {check.hint ? (
              <p className="doctor-stack__hint text-muted text-sm">{check.hint}</p>
            ) : null}
          </div>
        ))}
      </div>
    </div>
  );
}

export function DoctorPanel({ doctor, services, loading = false }: DoctorPanelProps) {
  const serviceList = services ?? [];
  const allChecks = doctor?.checks ?? [];
  const pinnedIds = new Set<string>(PINNED_CHECK_IDS);
  const checks = allChecks.filter((c) => !pinnedIds.has(c.id ?? ''));

  const serviceRows: StackRow[] = serviceList.map((svc) => ({
    title: displayLabel(svc.name),
    detail: humanizeTechnicalDetail(svc.detail),
    status: svc.status,
  }));

  const checkRows: StackRow[] = checks.map((check) => ({
    title: displayLabel(check.id),
    detail: humanizeTechnicalDetail(check.message),
    status: check.status,
  }));

  return (
    <section className="doctor-panel">
      <div className="doctor-panel__header">
        <h2 className="doctor-panel__title">Doctor</h2>
        {loading ? (
          <span className="text-muted text-sm">Loading...</span>
        ) : doctor?.overall ? (
          <StatusBadge status={doctor.overall} kind="service" />
        ) : null}
      </div>
      {allChecks.length > 0 ? <PinnedChecks checks={allChecks} /> : null}
      <StackSection label="" rows={serviceRows} />
      <StackSection label="Checks" rows={checkRows} />
      {!loading && serviceList.length === 0 && allChecks.length === 0 ? (
        <p className="doctor-panel__empty text-muted">No health data yet.</p>
      ) : null}
    </section>
  );
}
