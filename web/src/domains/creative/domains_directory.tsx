import { useEffect, useState } from 'react';

import { PrimaryActionButton, SecondaryActionButton } from '@/components/system/action_buttons';
import { PageChrome } from '@/components/system/page_chrome';
import { RowActionsMenu } from '@/components/system/row_actions_menu';
import { EmptyState } from '@/components/system/empty_state';
import { PageSkeleton } from '@/components/system/page_skeleton';
import { Badge } from '@/components/ui/badge';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { DropdownMenuItem } from '@/components/ui/dropdown-menu';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import type { DomainHealth, DomainSSLSetupResult } from '@/api/types';
import { CreativeNav, creativePanelError } from '@/domains/creative/creative_nav';
import { JsonDashboardView } from '@/domains/dashboards/json_dashboard_view';
import { displayTimestamp } from '@/lib/display';

export type DomainsDirectoryProps = {
  items: DomainHealth[];
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  draftHostname: string;
  acting: boolean;
  actionError: Error | undefined;
  actionMessage: string | undefined;
  sslResult: DomainSSLSetupResult | undefined;
  onDraftHostnameChange: (value: string) => void;
  onAddDomain: () => void;
  onDeleteDomain: (hostname: string) => void;
  onProbeDomain: (hostname: string) => void;
  onSetupSsl: (hostname: string) => void;
  draftParkDomain: string;
  draftParkZoneId: string;
  onDraftParkDomainChange: (value: string) => void;
  onDraftParkZoneIdChange: (value: string) => void;
  onParkDomain: () => void;
  parkMessage: string | undefined;
};

function healthBadgeVariant(
  status: DomainHealth['health_status'],
): 'default' | 'secondary' | 'destructive' | 'outline' {
  if (status === 'healthy') {
    return 'default';
  }
  if (status === 'degraded') {
    return 'secondary';
  }
  if (status === 'down') {
    return 'destructive';
  }
  return 'outline';
}

export function DomainsDirectory({
  items,
  fetching,
  error,
  hasSnapshot,
  draftHostname,
  acting,
  actionError,
  actionMessage,
  sslResult,
  onDraftHostnameChange,
  onAddDomain,
  onDeleteDomain,
  onProbeDomain,
  onSetupSsl,
  draftParkDomain,
  draftParkZoneId,
  onDraftParkDomainChange,
  onDraftParkZoneIdChange,
  onParkDomain,
  parkMessage,
}: DomainsDirectoryProps) {
  const [registerOpen, setRegisterOpen] = useState(false);
  const [parkOpen, setParkOpen] = useState(false);

  useEffect(() => {
    if (actionMessage === 'Domain registered') {
      setRegisterOpen(false);
    }
  }, [actionMessage]);

  useEffect(() => {
    if (parkMessage) {
      setParkOpen(false);
    }
  }, [parkMessage]);

  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton />;
  }

  if (error && !hasSnapshot) {
    return (
      <PageChrome title="Domains">
        <CreativeNav />
        {creativePanelError(error, 'Could not load domains')}
      </PageChrome>
    );
  }

  return (
    <PageChrome
      title="Domains"
      actions={
        <>
          <PrimaryActionButton onClick={() => setRegisterOpen(true)} type="button">
            Register domain
          </PrimaryActionButton>
          <SecondaryActionButton onClick={() => setParkOpen(true)} type="button">
            Park domain
          </SecondaryActionButton>
        </>
      }
    >
      <CreativeNav />

      <Dialog onOpenChange={setRegisterOpen} open={registerOpen}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>Register domain</DialogTitle>
          </DialogHeader>
          <div className="grid gap-2">
            <Label htmlFor="domain-hostname">Hostname</Label>
            <Input
              id="domain-hostname"
              value={draftHostname}
              onChange={(event) => onDraftHostnameChange(event.target.value)}
            />
          </div>
          <DialogFooter>
            <PrimaryActionButton loading={acting} onClick={onAddDomain} type="button">
              Add domain
            </PrimaryActionButton>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog onOpenChange={setParkOpen} open={parkOpen}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>Park domain</DialogTitle>
          </DialogHeader>
          <div className="grid grid-cols-[repeat(auto-fill,minmax(12rem,1fr))] gap-4">
            <div className="grid gap-2">
              <Label htmlFor="park-domain">Domain</Label>
              <Input
                id="park-domain"
                value={draftParkDomain}
                onChange={(event) => onDraftParkDomainChange(event.target.value)}
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="park-zone-id">Cloudflare zone ID</Label>
              <Input
                id="park-zone-id"
                value={draftParkZoneId}
                onChange={(event) => onDraftParkZoneIdChange(event.target.value)}
              />
            </div>
          </div>
          <DialogFooter>
            <SecondaryActionButton loading={acting} onClick={onParkDomain} type="button" variant="secondary">
              Park domain
            </SecondaryActionButton>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {items.length === 0 ? (
        <EmptyState title="No domains" description="Domain health list returned no entries." />
      ) : (
        <div className="ui-table-frame">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Hostname</TableHead>
                <TableHead>Role</TableHead>
                <TableHead>Health</TableHead>
                <TableHead>SSL</TableHead>
                <TableHead>Last probe</TableHead>
                <TableHead />
              </TableRow>
            </TableHeader>
            <TableBody>
              {items.map((row) => (
                <TableRow key={row.hostname}>
                  <TableCell>{row.hostname}</TableCell>
                  <TableCell>{row.role}</TableCell>
                  <TableCell>
                    <Badge variant={healthBadgeVariant(row.health_status)}>
                      {row.health_status}
                    </Badge>
                  </TableCell>
                  <TableCell>{row.ssl_status}</TableCell>
                  <TableCell>{displayTimestamp(row.last_probe_at)}</TableCell>
                  <TableCell>
                    <RowActionsMenu ariaLabel="Domain actions" disabled={acting}>
                      <DropdownMenuItem disabled={acting} onClick={() => onProbeDomain(row.hostname)}>
                        Probe
                      </DropdownMenuItem>
                      <DropdownMenuItem disabled={acting} onClick={() => onSetupSsl(row.hostname)}>
                        SSL setup
                      </DropdownMenuItem>
                      <DropdownMenuItem
                        className="text-destructive focus:text-destructive"
                        disabled={acting}
                        onClick={() => onDeleteDomain(row.hostname)}
                      >
                        Delete
                      </DropdownMenuItem>
                    </RowActionsMenu>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}

      {actionMessage ? (
        <p className="text-sm text-muted-foreground">{actionMessage}</p>
      ) : null}
      {sslResult ? (
        <JsonDashboardView payload={sslResult as unknown as Record<string, unknown>} />
      ) : null}
      {actionError ? creativePanelError(actionError, 'Domain action failed') : null}
      {error && hasSnapshot ? creativePanelError(error, 'Refresh failed') : null}
    </PageChrome>
  );
}
