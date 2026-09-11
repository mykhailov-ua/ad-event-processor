import { useCallback, useEffect, useState } from 'react';
import { toast } from 'sonner';

import { ackReportExportNotification, listReportExportNotifications } from '@/api/reports_api';
import type { ReportExportNotification } from '@/api/types';
import { useResource } from '@/api/use_resource';
import { useSession } from '@/hooks/use_session';
import { Button } from '@/components/ui/button';
import { exportHubErrorMessage } from '@/domains/exports/export_hub_errors';
import { adminTypography } from '@/lib/admin_kit';
import { sessionHasPermission } from '@/lib/session_permissions';
import { BentoSection } from '@/shell/bento_card';
import { ErrorBlock } from '@/shell/error_block';

const NOTIFICATION_POLL_MS = 15000;

export type ExportHubNotificationsProps = {
  onOpenJob?: (jobId: string) => void;
};

export function ExportHubNotifications({ onOpenJob }: ExportHubNotificationsProps) {
  const { user } = useSession();
  const canViewNotifications = sessionHasPermission(user?.permissions, 'exports:read');
  const [open, setOpen] = useState(false);
  const [pollToken, setPollToken] = useState(0);

  useEffect(() => {
    if (!canViewNotifications) {
      return;
    }
    const timer = window.setInterval(() => {
      setPollToken((value) => value + 1);
    }, NOTIFICATION_POLL_MS);
    return () => {
      window.clearInterval(timer);
    };
  }, [canViewNotifications]);

  const { data, error, revalidating } = useResource(
    (signal) => {
      if (!canViewNotifications) {
        return Promise.resolve(undefined);
      }
      return listReportExportNotifications(signal);
    },
    [pollToken, open, canViewNotifications]
  );

  const onAck = useCallback(
    async (notification: ReportExportNotification) => {
      if (!notification.id) {
        return;
      }
      try {
        await ackReportExportNotification(notification.id);
        if (onOpenJob && notification.job_id) {
          onOpenJob(notification.job_id);
        }
        setOpen(false);
        setPollToken((value) => value + 1);
      } catch (err: unknown) {
        toast.error(exportHubErrorMessage(err, 'Failed to acknowledge notification'));
      }
    },
    [onOpenJob]
  );

  if (!canViewNotifications) {
    return null;
  }

  const unreadCount = data?.unread_count ?? 0;
  const rows = data?.rows ?? [];

  return (
    <div className="relative">
      <Button
        aria-expanded={open}
        aria-haspopup="dialog"
        type="button"
        variant="outline"
        onClick={() => setOpen((value) => !value)}
      >
        Notifications{unreadCount > 0 ? ` (${unreadCount})` : ''}
      </Button>
      {open ? (
        <div className="absolute right-0 top-full z-20 mt-2 w-[min(28rem,calc(100vw-2rem))] border bg-background p-3 shadow-md">
          <BentoSection
            data-testid="export-hub-notifications"
            title="Export notifications"
          >
            {error ? (
              <ErrorBlock error={error} title="Could not load export notifications" />
            ) : null}
            {revalidating && rows.length === 0 && !error ? (
              <p className={adminTypography.bodyMuted}>Loading notifications...</p>
            ) : null}
            {!error && rows.length === 0 && !revalidating ? (
              <p className={adminTypography.bodyMuted}>No export notifications yet.</p>
            ) : null}
            <ul data-role="notification-list">
              {rows.map((row) => (
                <li key={row.id} data-role="notification-item">
                  <Button
                    className="h-auto min-h-0 w-full justify-start whitespace-normal px-0 py-2 text-left font-normal hover:bg-transparent"
                    type="button"
                    variant="ghost"
                    onClick={() => void onAck(row)}
                  >
                    <p className={adminTypography.label}>{row.title}</p>
                    <p className={adminTypography.bodyMuted}>{row.body}</p>
                    <p className={adminTypography.monoData}>
                      Job {row.job_id}
                      {!row.read ? ' / unread' : ''}
                    </p>
                  </Button>
                </li>
              ))}
            </ul>
          </BentoSection>
        </div>
      ) : null}
    </div>
  );
}
