import { useMemo, useState } from 'react';
import { createSortState, toggleSort } from '../lib/table_sort.js';
import * as storage from '../helpers/storage.js';
import { AlertBanner } from '../components/alert_banner.js';
import { Button } from '../components/button.js';
import { CampaignWizard } from '../components/campaign_wizard.js';
import { DataTable } from '../components/data_table.js';
import { DatePicker, formatDisplayDateTime } from '../components/date_picker.js';
import { EmptyState } from '../components/empty_state.js';
import { FormField } from '../components/form_field.js';
import { Modal } from '../components/modal.js';
import { PageHeader } from '../components/page_header.js';
import { PageSkeleton } from '../components/page_skeleton.js';
import { PageStack } from '../components/page_stack.js';
import { SectionCard } from '../components/section_card.js';
import { StatusBadge } from '../components/status_badge.js';
import { useToast } from '../helpers/use_toast.js';

type DemoRow = {
  id: string;
  name: string;
  status: string;
  spend: number;
};

const DEMO_ROWS: DemoRow[] = [
  { id: '1', name: 'Alpha campaign', status: 'active', spend: 1200 },
  { id: '2', name: 'Beta campaign', status: 'paused', spend: 450 },
  { id: '3', name: 'Gamma campaign', status: 'archived', spend: 90 },
];

export function DevComponentsPage() {
  const pushToast = useToast();
  const [modalOpen, setModalOpen] = useState(false);
  const [fieldValue, setFieldValue] = useState('');
  const [fieldError, setFieldError] = useState<string | null>(null);
  const [theme, setTheme] = useState(storage.getTheme());
  const [sortState, setSortState] = useState(() => createSortState('name', 'asc'));
  const [reportFrom, setReportFrom] = useState(() => new Date().toISOString());
  const [wizardOpen, setWizardOpen] = useState(false);

  const columns = useMemo(
    () => [
      {
        key: 'name',
        header: 'Name',
        sortable: true,
        accessor: (row: DemoRow) => row.name,
        render: (row: DemoRow) => row.name,
      },
      {
        key: 'status',
        header: 'Status',
        sortable: true,
        accessor: (row: DemoRow) => row.status,
        render: (row: DemoRow) => <StatusBadge status={row.status} kind="campaign" />,
      },
      {
        key: 'spend',
        header: 'Spend',
        sortable: true,
        accessor: (row: DemoRow) => row.spend,
        render: (row: DemoRow) => row.spend.toLocaleString('en-US'),
      },
    ],
    []
  );

  const toggleTheme = () => {
    const next = theme === 'dark' ? 'light' : 'dark';
    storage.setTheme(next);
    document.documentElement.setAttribute('data-theme', next);
    setTheme(next);
  };

  const validateField = () => {
    if (!fieldValue.trim()) {
      setFieldError('Campaign name is required.');
      return;
    }
    setFieldError(null);
    pushToast('Saved', 'Form field validation passed.');
  };

  return (
    <div className="dev-gallery">
      <h1>UI component gallery</h1>

      <SectionCard title="Buttons" desc="Primary, secondary, danger, and ghost variants.">
        <div className="cluster cluster--actions">
          <Button
            label="Primary"
            variant="primary"
            onClick={() => pushToast('Primary', 'Primary button clicked.')}
          />
          <Button label="Secondary" variant="secondary" />
          <Button label="Danger" variant="danger" />
          <Button label="Ghost" variant="ghost" />
          <Button label="Loading" variant="primary" loading />
        </div>
      </SectionCard>

      <SectionCard title="Form field" desc="Label, hint, error, and aria-invalid wiring.">
        <FormField
          label="Campaign name"
          hint="Shown when the field is valid."
          error={fieldError}
          reserveFeedback
        >
          <input
            className="input"
            value={fieldValue}
            onChange={(e) => {
              setFieldValue(e.target.value);
              if (fieldError) setFieldError(null);
            }}
          />
        </FormField>
        <div className="cluster cluster--actions mt-4">
          <Button label="Validate" variant="secondary" size="sm" onClick={validateField} />
        </div>
      </SectionCard>

      <SectionCard title="Data table" desc="Sortable headers with status badges.">
        <DataTable
          caption="Demo campaigns"
          columns={columns}
          rows={DEMO_ROWS}
          sortState={sortState}
          onSort={(key) => {
            setSortState((prev) => {
              const next = { ...prev };
              toggleSort(next, key);
              return next;
            });
          }}
          rowKey={(row) => row.id}
        />
      </SectionCard>

      <SectionCard title="Modal" desc="Focus trap, escape, and overlay dismiss.">
        <Button label="Open modal" variant="secondary" onClick={() => setModalOpen(true)} />
        <Modal
          open={modalOpen}
          title="Confirm sample action"
          description="This modal uses the shared React wrapper."
          onClose={() => setModalOpen(false)}
          actions={
            <>
              <Button label="Cancel" variant="ghost" onClick={() => setModalOpen(false)} />
              <Button
                label="Confirm"
                variant="primary"
                onClick={() => {
                  setModalOpen(false);
                  pushToast('Confirmed', 'Modal primary action fired.');
                }}
              />
            </>
          }
        />
      </SectionCard>

      <SectionCard title="Alert banners" desc="Info, warning, and error tones.">
        <AlertBanner variant="info" message="Info alert — check the ops dashboard." />
        <AlertBanner variant="warning" message="Warning alert — pacing drift detected." />
        <AlertBanner variant="error" message="Error alert — export job failed." />
      </SectionCard>

      <SectionCard title="Status badges" desc="Campaign, service, and invoice mappings.">
        <div className="cluster">
          <StatusBadge status="active" kind="campaign" />
          <StatusBadge status="paused" kind="campaign" />
          <StatusBadge status="ok" kind="service" label="Healthy" />
          <StatusBadge status="paid" kind="invoice" />
          <StatusBadge status="void" kind="invoice" />
        </div>
      </SectionCard>

      <SectionCard title="Toast" desc="Uses the imperative toast stack from AppShell.">
        <Button
          label="Show toast"
          variant="secondary"
          onClick={() => pushToast('Notification', 'Toast integration is wired.', 'DEMO_TOAST')}
        />
      </SectionCard>

      <SectionCard title="Date picker" desc="Popover calendar + time selects; ISO value in state.">
        <FormField
          label="Report window start"
          hint={`Selected: ${formatDisplayDateTime(new Date(reportFrom)) || '—'}`}
        >
          <DatePicker id="dev-report-from" value={reportFrom} onChange={setReportFrom} />
        </FormField>
      </SectionCard>

      <SectionCard
        title="Campaign wizard"
        desc="Create flow modal (API call only when opened with a real customer UUID)."
      >
        <Button
          label="Open wizard (demo)"
          variant="secondary"
          onClick={() => setWizardOpen(true)}
        />
        <CampaignWizard
          open={wizardOpen}
          customerId="00000000-0000-4000-8000-000000000001"
          onClose={() => setWizardOpen(false)}
          onCreated={(id) => {
            setWizardOpen(false);
            pushToast('Wizard', `onCreated fired: ${id}`);
          }}
        />
      </SectionCard>

      <SectionCard
        title="Theme"
        desc={`Current theme: ${theme}. Toggle uses storage.getTheme / setTheme.`}
      >
        <Button
          label={theme === 'dark' ? 'Switch to light' : 'Switch to dark'}
          variant="secondary"
          onClick={toggleTheme}
        />
      </SectionCard>

      <SectionCard
        title="PageHeader"
        desc="Standard page header with breadcrumbs, title, description, and actions slot."
      >
        <div
          style={{
            background: 'var(--bg-canvas)',
            padding: 'var(--space-md)',
            borderRadius: 'var(--radius-card)',
          }}
        >
          <PageHeader
            title="Campaign detail"
            desc="Configure targeting, pacing, and creative assets."
            breadcrumbs={[{ label: 'Campaigns', to: '/campaigns' }, { label: 'Alpha Q3 Push' }]}
            badge={<StatusBadge status="active" kind="campaign" />}
            actions={
              <>
                <Button label="Edit" variant="secondary" size="sm" />
                <Button label="Pause" variant="primary" size="sm" />
              </>
            }
          />
        </div>
        <p className="text-hint">Compact variant (for sub-sections):</p>
        <div
          style={{
            background: 'var(--bg-canvas)',
            padding: 'var(--space-md)',
            borderRadius: 'var(--radius-card)',
          }}
        >
          <PageHeader title="Targeting rules" compact />
        </div>
      </SectionCard>

      <SectionCard
        title="PageStack"
        desc="Enforces vertical rhythm via the .stack utility. Compare with raw div nesting."
      >
        <PageStack gap="md">
          <div className="settings-panel">
            <p className="text-hint">Section A inside PageStack gap=md</p>
          </div>
          <div className="settings-panel">
            <p className="text-hint">Section B — consistent gap from PageStack</p>
          </div>
        </PageStack>
      </SectionCard>

      <SectionCard
        title="PageSkeleton"
        desc="Loading shimmer that preserves spatial layout during data fetch."
      >
        <PageSkeleton rows={2} />
      </SectionCard>

      <SectionCard
        title="EmptyState"
        desc="Standardized empty state for tables and panels. One title, one desc, one CTA."
      >
        <EmptyState
          icon="funnel"
          title="No campaigns match your filters"
          desc="Clear the active filters or create a new campaign."
          action={<Button label="Clear filters" variant="secondary" size="sm" />}
        />
      </SectionCard>
    </div>
  );
}
