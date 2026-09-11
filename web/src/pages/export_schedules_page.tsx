import { ExportSchedulesPageView } from '@/domains/exports/export_schedules_page_view';
import { useExportSchedulesPageWorkspace } from '@/domains/exports/use_export_schedules_page_workspace';

export function ExportSchedulesPage() {
  const workspace = useExportSchedulesPageWorkspace();
  return (
    <ExportSchedulesPageView
      canManage={workspace.canManage}
      customerId={workspace.customerId}
      draft={workspace.draft}
      formValidationError={workspace.formValidationError}
      lastJobHref={workspace.lastJobHref}
      runningScheduleId={workspace.runningScheduleId}
      saving={workspace.saving}
      schedules={workspace.schedules}
      schedulesError={workspace.schedulesError}
      schedulesFetching={workspace.schedulesFetching}
      onCustomerIdChange={workspace.setCustomerId}
      onDeleteSchedule={workspace.onDeleteSchedule}
      onDraftChange={(patch) => workspace.setDraft({ ...workspace.draft, ...patch })}
      onResetDraft={workspace.resetDraft}
      onRunScheduleNow={workspace.onRunScheduleNow}
      onSaveSchedule={workspace.onSaveSchedule}
      onSelectSchedule={workspace.loadScheduleIntoDraft}
      onToggleSchedule={workspace.onToggleSchedule}
    />
  );
}
