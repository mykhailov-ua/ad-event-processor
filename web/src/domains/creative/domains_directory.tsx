import { useState } from 'react';
import { useRunWhenTrue } from '@/hooks/use_run_when_true';

import { PrimaryActionButton, SecondaryActionButton, FilterApplyButton } from '@/shell/action_buttons';
import { DirectoryFilterForm, FilterField } from '@/shell/filter_panel';
import { PageChrome } from '@/shell/page_chrome';
import { RowActionsMenu } from '@/shell/row_actions_menu';
import { EmptyState } from '@/shell/empty_state';
import { PageSkeleton } from '@/shell/page_skeleton';
import { Checkbox } from '@/components/ui/checkbox';
import { Label } from '@/components/ui/label';
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  DirectoryTable,
  DirectoryTableHead,
  TableBody,
  TableCell,
  TableHeader,
  TableRow,
} from '@/shell/directory_table';
import { ErrorBlock } from '@/shell/error_block';
import type {
  CloudflareZone,
  DomainBulkJobStatus,
  DomainBulkJobRow,
  DomainHealth,
  DomainSSLSetupResult,
  WildcardSSLResponse,
} from '@/api/types';
import type { DomainHealthFilter } from '@/domains/creative/use_domains_page_workspace';
import { CreativeDirectoryStack } from '@/domains/creative/creative_directory_stack';
import { creativePanelError } from '@/domains/creative/creative_nav';
import { TableHost } from '@/shell/ui_bands';
import { displayTimestamp } from '@/lib/display';

export type DomainsDirectoryProps = {
  items?: DomainHealth[];
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
  onBurnDomain: (hostname: string) => void;
  burnOpen: boolean;
  onBurnOpenChange: (open: boolean) => void;
  burnHostname: string;
  burnDeleteCloudflare: boolean;
  onBurnDeleteCloudflareChange: (value: boolean) => void;
  onConfirmBurn: () => void;
  onProbeDomain: (hostname: string) => void;
  onSetupSsl: (hostname: string) => void;
  draftParkDomain: string;
  draftParkZoneId: string;
  onDraftParkDomainChange: (value: string) => void;
  onDraftParkZoneIdChange: (value: string) => void;
  onParkDomain: () => void;
  parkMessage: string | undefined;
  draftHealthFilter: DomainHealthFilter;
  onDraftHealthFilterChange: (value: DomainHealthFilter) => void;
  onApplyHealthFilter: (event?: { preventDefault?: () => void }) => void;
  bulkOpen: boolean;
  onBulkOpenChange: (open: boolean) => void;
  onOpenBulkDialog: () => void;
  draftBulkText: string;
  onDraftBulkTextChange: (value: string) => void;
  draftBulkCsvLoaded: boolean;
  onBulkCsvFileSelected: (file: File | undefined) => void;
  draftBulkZoneId: string;
  onDraftBulkZoneIdChange: (value: string) => void;
  onStartBulkPark: () => void;
  onStartBulkSSL: () => void;
  bulkJob: DomainBulkJobStatus | undefined;
  bulkJobError: Error | undefined;
  wildcardOpen: boolean;
  onWildcardOpenChange: (open: boolean) => void;
  onOpenWildcardWizard: () => void;
  cloudflareZones: CloudflareZone[];
  zonesError: Error | undefined;
  draftWildcardZoneId: string;
  draftWildcardZoneName: string;
  draftWildcardIncludeApex: boolean;
  onDraftWildcardZoneIdChange: (value: string) => void;
  onDraftWildcardZoneNameChange: (value: string) => void;
  onDraftWildcardIncludeApexChange: (value: boolean) => void;
  onSetupWildcardSSL: () => void;
  wildcardResult: WildcardSSLResponse | undefined;
};

function healthBadgeVariant(
  status: DomainHealth['health_status']
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

function DomainSslResultSummary({ result }: { result: DomainSSLSetupResult }) {
  return (
    <div className="rounded-md border p-3 text-sm">
      <p>
        <span className="font-medium">{result.hostname}</span>
        <span className="text-muted-foreground"> | {result.status}</span>
      </p>
      <p className="text-muted-foreground">{result.message}</p>
      {result.output ? (
        <pre className="mt-2 max-h-40 overflow-auto whitespace-pre-wrap text-xs">
          {result.output}
        </pre>
      ) : null}
    </div>
  );
}

function WildcardSslResultSummary({ result }: { result: WildcardSSLResponse }) {
  return (
    <div className="grid gap-1 rounded-md border p-3 text-sm">
      <p className="font-medium">{result.wildcard_hostname}</p>
      <p className="text-muted-foreground">
        ACME {result.acme_state} | pool {result.pool_id}
        {result.ssl_not_after ? ` | expires ${displayTimestamp(result.ssl_not_after)}` : ''}
        {' | '}
        CF proxied {result.cloudflare_proxied ? 'yes' : 'no'}
      </p>
      {result.message ? <p className="text-muted-foreground">{result.message}</p> : null}
    </div>
  );
}

function DomainBulkJobResultsTable({ rows }: { rows: DomainBulkJobRow[] }) {
  if (rows.length === 0) {
    return null;
  }
  return (
    <TableHost>
      <DirectoryTable nested>
        <TableHeader>
          <TableRow>
            <DirectoryTableHead>Hostname</DirectoryTableHead>
            <DirectoryTableHead>Result</DirectoryTableHead>
            <DirectoryTableHead>Error</DirectoryTableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.map((row) => (
            <TableRow key={row.hostname}>
              <TableCell>{row.hostname}</TableCell>
              <TableCell>
                <Badge variant={row.ok ? 'default' : 'destructive'}>
                  {row.ok ? 'ok' : 'failed'}
                </Badge>
              </TableCell>
              <TableCell className="text-muted-foreground">{row.error ?? ''}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </DirectoryTable>
    </TableHost>
  );
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
  onBurnDomain,
  burnOpen,
  onBurnOpenChange,
  burnHostname,
  burnDeleteCloudflare,
  onBurnDeleteCloudflareChange,
  onConfirmBurn,
  onProbeDomain,
  onSetupSsl,
  draftParkDomain,
  draftParkZoneId,
  onDraftParkDomainChange,
  onDraftParkZoneIdChange,
  onParkDomain,
  parkMessage,
  draftHealthFilter,
  onDraftHealthFilterChange,
  onApplyHealthFilter,
  bulkOpen,
  onBulkOpenChange,
  onOpenBulkDialog,
  draftBulkText,
  onDraftBulkTextChange,
  draftBulkCsvLoaded,
  onBulkCsvFileSelected,
  draftBulkZoneId,
  onDraftBulkZoneIdChange,
  onStartBulkPark,
  onStartBulkSSL,
  bulkJob,
  bulkJobError,
  wildcardOpen,
  onWildcardOpenChange,
  onOpenWildcardWizard,
  cloudflareZones,
  zonesError,
  draftWildcardZoneId,
  draftWildcardZoneName,
  draftWildcardIncludeApex,
  onDraftWildcardZoneIdChange,
  onDraftWildcardZoneNameChange,
  onDraftWildcardIncludeApexChange,
  onSetupWildcardSSL,
  wildcardResult,
}: DomainsDirectoryProps) {
  const [registerOpen, setRegisterOpen] = useState(false);
  const [parkOpen, setParkOpen] = useState(false);

  useRunWhenTrue(actionMessage === 'Domain registered', () => setRegisterOpen(false));
  useRunWhenTrue(Boolean(parkMessage), () => setParkOpen(false));

  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton />;
  }

  if (error && !hasSnapshot) {
    return (
      <PageChrome title="Domains">
        <CreativeDirectoryStack>
          {creativePanelError(error, 'Could not load domains')}
        </CreativeDirectoryStack>
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
          <SecondaryActionButton onClick={onOpenBulkDialog} type="button">
            Bulk import
          </SecondaryActionButton>
          <SecondaryActionButton onClick={() => setParkOpen(true)} type="button">
            Park domain
          </SecondaryActionButton>
          <SecondaryActionButton onClick={onOpenWildcardWizard} type="button">
            Wildcard SSL
          </SecondaryActionButton>
        </>
      }
    >
      <CreativeDirectoryStack>
        <DirectoryFilterForm layout="auto-fill" onSubmit={onApplyHealthFilter}>
          <FilterField htmlFor="domain-health-filter" label="Health filter">
            <Select
              value={draftHealthFilter}
              onValueChange={(value) => onDraftHealthFilterChange(value as DomainHealthFilter)}
            >
              <SelectTrigger id="domain-health-filter" className="w-full max-w-xs">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All</SelectItem>
                <SelectItem value="healthy">Healthy</SelectItem>
                <SelectItem value="degraded">Degraded</SelectItem>
                <SelectItem value="burned">Burned</SelectItem>
              </SelectContent>
            </Select>
          </FilterField>
          <FilterApplyButton disabled={fetching} type="submit">
            Apply
          </FilterApplyButton>
        </DirectoryFilterForm>

        <Dialog onOpenChange={onBulkOpenChange} open={bulkOpen}>
          <DialogContent className="max-w-lg">
            <DialogHeader>
              <DialogTitle>Bulk domain import</DialogTitle>
            </DialogHeader>
            {zonesError ? (
              <ErrorBlock title="Could not load zones" message={zonesError.message} />
            ) : null}
            <DirectoryFilterForm layout="auto-fill" onSubmit={(event) => event.preventDefault()}>
              <FilterField htmlFor="bulk-zone-picker" label="Cloudflare zone (park)">
                <Select value={draftBulkZoneId} onValueChange={onDraftBulkZoneIdChange}>
                  <SelectTrigger id="bulk-zone-picker" className="w-full">
                    <SelectValue placeholder="Select zone for park" />
                  </SelectTrigger>
                  <SelectContent>
                    {cloudflareZones.map((zone) => (
                      <SelectItem key={zone.id} value={zone.id}>
                        {zone.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </FilterField>
              <FilterField htmlFor="bulk-hostnames" label="Hostnames (one per line)">
                <textarea
                  className="min-h-32 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
                  id="bulk-hostnames"
                  onChange={(event) => onDraftBulkTextChange(event.target.value)}
                  value={draftBulkText}
                />
              </FilterField>
              <FilterField htmlFor="bulk-csv-file" label="Or upload CSV">
                <input
                  accept=".csv,text/csv,text/plain"
                  disabled={acting}
                  id="bulk-csv-file"
                  onChange={(event) => {
                    onBulkCsvFileSelected(event.target.files?.[0]);
                    event.target.value = '';
                  }}
                  type="file"
                />
                {draftBulkCsvLoaded ? (
                  <p className="text-sm text-muted-foreground" role="status">
                    CSV file loaded
                  </p>
                ) : null}
              </FilterField>
            </DirectoryFilterForm>
            {bulkJob ? (
              <p className="text-sm text-muted-foreground">
                Job {bulkJob.job_id}: {bulkJob.status} ({bulkJob.completed}/{bulkJob.total}
                {bulkJob.failed > 0 ? `, ${bulkJob.failed} failed` : ''})
              </p>
            ) : null}
            {bulkJob?.results && bulkJob.results.length > 0 ? (
              <DomainBulkJobResultsTable rows={bulkJob.results} />
            ) : null}
            {bulkJobError ? (
              <ErrorBlock title="Job poll failed" message={bulkJobError.message} />
            ) : null}
            <DialogFooter>
              <SecondaryActionButton loading={acting} onClick={onStartBulkSSL} type="button">
                Bulk SSL only
              </SecondaryActionButton>
              <PrimaryActionButton loading={acting} onClick={onStartBulkPark} type="button">
                Park + probe
              </PrimaryActionButton>
            </DialogFooter>
          </DialogContent>
        </Dialog>

        <Dialog onOpenChange={setRegisterOpen} open={registerOpen}>
          <DialogContent className="max-w-md">
            <DialogHeader>
              <DialogTitle>Register domain</DialogTitle>
            </DialogHeader>
            <FilterField htmlFor="domain-hostname" label="Hostname">
              <Input
                id="domain-hostname"
                value={draftHostname}
                onChange={(event) => onDraftHostnameChange(event.target.value)}
              />
            </FilterField>
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
            <DirectoryFilterForm layout="auto-fill" onSubmit={(event) => event.preventDefault()}>
              <FilterField htmlFor="park-domain" label="Domain">
                <Input
                  id="park-domain"
                  value={draftParkDomain}
                  onChange={(event) => onDraftParkDomainChange(event.target.value)}
                />
              </FilterField>
              <FilterField htmlFor="park-zone-id" label="Cloudflare zone ID">
                <Input
                  id="park-zone-id"
                  value={draftParkZoneId}
                  onChange={(event) => onDraftParkZoneIdChange(event.target.value)}
                />
              </FilterField>
            </DirectoryFilterForm>
            <DialogFooter>
              <SecondaryActionButton
                loading={acting}
                onClick={onParkDomain}
                type="button"
                variant="secondary"
              >
                Park domain
              </SecondaryActionButton>
            </DialogFooter>
          </DialogContent>
        </Dialog>

        <Dialog onOpenChange={onWildcardOpenChange} open={wildcardOpen}>
          <DialogContent className="max-w-lg">
            <DialogHeader>
              <DialogTitle>Wildcard SSL (DNS-01)</DialogTitle>
            </DialogHeader>
            {zonesError ? (
              <ErrorBlock title="Could not load zones" message={zonesError.message} />
            ) : null}
            <DirectoryFilterForm layout="auto-fill" onSubmit={(event) => event.preventDefault()}>
              <FilterField htmlFor="wildcard-zone-picker" label="Cloudflare zone">
                <Select
                  value={draftWildcardZoneId}
                  onValueChange={(zoneId) => {
                    onDraftWildcardZoneIdChange(zoneId);
                    const zone = cloudflareZones.find((row) => row.id === zoneId);
                    if (zone?.name) {
                      onDraftWildcardZoneNameChange(zone.name);
                    }
                  }}
                >
                  <SelectTrigger id="wildcard-zone-picker" className="w-full">
                    <SelectValue placeholder="Select zone" />
                  </SelectTrigger>
                  <SelectContent>
                    {cloudflareZones.map((zone) => (
                      <SelectItem key={zone.id} value={zone.id}>
                        {zone.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </FilterField>
              <FilterField htmlFor="wildcard-zone-name" label="Zone name">
                <Input
                  id="wildcard-zone-name"
                  value={draftWildcardZoneName}
                  onChange={(event) => onDraftWildcardZoneNameChange(event.target.value)}
                  placeholder="trk.example.com"
                />
              </FilterField>
              <div className="flex items-center gap-2">
                <Checkbox
                  checked={draftWildcardIncludeApex}
                  id="wildcard-include-apex"
                  onCheckedChange={(checked) => onDraftWildcardIncludeApexChange(checked === true)}
                />
                <Label htmlFor="wildcard-include-apex">Include apex in certificate</Label>
              </div>
            </DirectoryFilterForm>
            <DialogFooter>
              <PrimaryActionButton loading={acting} onClick={onSetupWildcardSSL} type="button">
                Issue wildcard cert
              </PrimaryActionButton>
            </DialogFooter>
          </DialogContent>
        </Dialog>

        <Dialog onOpenChange={onBurnOpenChange} open={burnOpen}>
          <DialogContent className="max-w-md">
            <DialogHeader>
              <DialogTitle>Burn domain</DialogTitle>
            </DialogHeader>
            <p className="text-sm text-muted-foreground">
              Burn <span className="font-medium text-foreground">{burnHostname}</span> and remove it
              from the rotation pool.
            </p>
            <div className="flex items-center gap-2">
              <Checkbox
                checked={burnDeleteCloudflare}
                id="burn-delete-cloudflare"
                onCheckedChange={(checked) => onBurnDeleteCloudflareChange(checked === true)}
              />
              <Label htmlFor="burn-delete-cloudflare">Also delete Cloudflare DNS record</Label>
            </div>
            <DialogFooter>
              <SecondaryActionButton onClick={() => onBurnOpenChange(false)} type="button">
                Cancel
              </SecondaryActionButton>
              <PrimaryActionButton loading={acting} onClick={onConfirmBurn} type="button">
                Confirm burn
              </PrimaryActionButton>
            </DialogFooter>
          </DialogContent>
        </Dialog>

        {(items ?? []).length === 0 ? (
          <EmptyState title="No domains" description="Domain health list returned no entries." />
        ) : (
          <TableHost>
            <DirectoryTable nested>
              <TableHeader>
                <TableRow>
                  <DirectoryTableHead>Hostname</DirectoryTableHead>
                  <DirectoryTableHead>Role</DirectoryTableHead>
                  <DirectoryTableHead>Health</DirectoryTableHead>
                  <DirectoryTableHead>SSL</DirectoryTableHead>
                  <DirectoryTableHead>SSL expiry</DirectoryTableHead>
                  <DirectoryTableHead>ACME</DirectoryTableHead>
                  <DirectoryTableHead>CF proxied</DirectoryTableHead>
                  <DirectoryTableHead>Pool</DirectoryTableHead>
                  <DirectoryTableHead>Last probe</DirectoryTableHead>
                  <DirectoryTableHead />
                </TableRow>
              </TableHeader>
              <TableBody>
                {(items ?? []).map((row) => (
                  <TableRow key={row.hostname}>
                    <TableCell>{row.hostname}</TableCell>
                    <TableCell>{row.role}</TableCell>
                    <TableCell>
                      <Badge variant={healthBadgeVariant(row.health_status)}>
                        {row.health_status}
                      </Badge>
                    </TableCell>
                    <TableCell>{row.ssl_status}</TableCell>
                    <TableCell>{displayTimestamp(row.ssl_not_after)}</TableCell>
                    <TableCell>{row.acme_state ?? '-'}</TableCell>
                    <TableCell>{row.cloudflare_proxied ? 'yes' : 'no'}</TableCell>
                    <TableCell>{row.pool_status ?? '-'}</TableCell>
                    <TableCell>{displayTimestamp(row.last_probe_at)}</TableCell>
                    <TableCell>
                      <RowActionsMenu ariaLabel="Domain actions" disabled={acting}>
                        <DropdownMenuItem
                          disabled={acting}
                          onClick={() => onProbeDomain(row.hostname)}
                        >
                          Probe
                        </DropdownMenuItem>
                        <DropdownMenuItem
                          disabled={acting}
                          onClick={() => onSetupSsl(row.hostname)}
                        >
                          SSL setup
                        </DropdownMenuItem>
                        {row.pool_status !== 'banned' ? (
                          <DropdownMenuItem
                            className="text-destructive focus:text-destructive"
                            disabled={acting}
                            onClick={() => onBurnDomain(row.hostname)}
                          >
                            Burn
                          </DropdownMenuItem>
                        ) : null}
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
            </DirectoryTable>
          </TableHost>
        )}

        {actionMessage ? <p className="text-sm text-muted-foreground">{actionMessage}</p> : null}
        {sslResult ? <DomainSslResultSummary result={sslResult} /> : null}
        {wildcardResult ? <WildcardSslResultSummary result={wildcardResult} /> : null}
        {actionError ? creativePanelError(actionError, 'Domain action failed') : null}
        {error && hasSnapshot ? creativePanelError(error, 'Refresh failed') : null}
      </CreativeDirectoryStack>
    </PageChrome>
  );
}
