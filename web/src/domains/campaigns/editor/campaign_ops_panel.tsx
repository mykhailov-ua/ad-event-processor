import { campaignPanelError } from '@/domains/campaigns/editor/campaign_editor_shared';
import {
  FilterField,
  FILTER_PANEL_SUMMARY_CLASS,
  INLINE_FILTER_ACTION_GRID_CLASS,
} from '@/shell/filter_panel';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  DirectoryTable,
  DirectoryTableHead,
  TableBody,
  TableCell,
  TableHeader,
  TableRow,
} from '@/shell/directory_table';
import { displayTimestamp } from '@/lib/display';
import type { CampaignOpsPanelWorkspace } from '@/domains/campaigns/editor/use_campaign_ops_panel_workspace';

export type CampaignOpsPanelProps = {
  workspace: CampaignOpsPanelWorkspace;
};

export function CampaignOpsPanel({ workspace }: CampaignOpsPanelProps) {
  const {
    draftPlacementId,
    setDraftPlacementId,
    loadingKey,
    stats,
    events,
    margin,
    mappings,
    mappingDrafts,
    setMappingDrafts,
    suggestions,
    smokeMessage,
    flowMessage,
    actionError,
    savingMappings,
    mappingSaveSuccess,
    busy,
    blocking,
    onLoadStats,
    onLoadEvents,
    onLoadMargin,
    onLoadMappings,
    onLoadSuggestions,
    onRunSmoke,
    onValidateFlow,
    onSaveMappings,
    onBlockPlacement,
  } = workspace;

  return (
    <div className="grid gap-4">
      <div className="flex flex-wrap gap-2">
        <Button disabled={busy} onClick={onLoadStats} type="button" variant="outline">
          {loadingKey === 'stats' ? 'Loading...' : 'Stats'}
        </Button>
        <Button disabled={busy} onClick={onLoadEvents} type="button" variant="outline">
          {loadingKey === 'events' ? 'Loading...' : 'Events'}
        </Button>
        <Button disabled={busy} onClick={onLoadMargin} type="button" variant="outline">
          {loadingKey === 'margin' ? 'Loading...' : 'Margin'}
        </Button>
        <Button disabled={busy} onClick={onLoadMappings} type="button" variant="outline">
          {loadingKey === 'mappings' ? 'Loading...' : 'Conversion mappings'}
        </Button>
        <Button disabled={busy} onClick={onLoadSuggestions} type="button" variant="outline">
          {loadingKey === 'suggestions' ? 'Loading...' : 'Placement suggestions'}
        </Button>
        <Button disabled={busy} onClick={onRunSmoke} type="button" variant="secondary">
          {loadingKey === 'smoke' ? 'Running...' : 'Smoke test'}
        </Button>
        <Button disabled={busy} onClick={onValidateFlow} type="button" variant="secondary">
          {loadingKey === 'flow' ? 'Validating...' : 'Validate flow'}
        </Button>
      </div>

      {stats ? (
        <section className="ui-filter-panel gap-2 text-sm">
          <h3 className="font-semibold">Campaign stats</h3>
          <div className="grid grid-cols-[repeat(auto-fill,minmax(10rem,1fr))] gap-2">
            <div>
              <span className="text-muted-foreground">Current spend</span>
              <p className="tabular-nums">{stats.current_spend ?? ''}</p>
            </div>
            <div>
              <span className="text-muted-foreground">Clicks</span>
              <p className="tabular-nums">{stats.metrics?.clicks ?? 0}</p>
            </div>
            <div>
              <span className="text-muted-foreground">Conversions</span>
              <p className="tabular-nums">{stats.metrics?.conversions ?? 0}</p>
            </div>
            <div>
              <span className="text-muted-foreground">Impressions</span>
              <p className="tabular-nums">{stats.metrics?.impressions ?? 0}</p>
            </div>
          </div>
        </section>
      ) : null}

      {margin ? (
        <section className={FILTER_PANEL_SUMMARY_CLASS}>
          <h3 className="font-semibold">Margin</h3>
          <p>
            Operator margin (micro): <strong>{margin.operator_margin_micro ?? ''}</strong>
          </p>
          <p>
            Advertiser spend (micro): <strong>{margin.advertiser_spend_micro ?? ''}</strong>
          </p>
          <p>
            Margin breach: <strong>{margin.margin_breach ? 'yes' : 'no'}</strong>
          </p>
        </section>
      ) : null}

      {events && (events.items?.length ?? 0) > 0 ? (
        <DirectoryTable>
          <TableHeader>
            <TableRow>
              <DirectoryTableHead>Time</DirectoryTableHead>
              <DirectoryTableHead>Type</DirectoryTableHead>
              <DirectoryTableHead>Click ID</DirectoryTableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {events.items?.map((row, index) => (
              <TableRow key={`${row.click_id ?? 'event'}-${index}`}>
                <TableCell>{displayTimestamp(row.created_at)}</TableCell>
                <TableCell>{row.event_type ?? ''}</TableCell>
                <TableCell className="font-mono text-xs">{row.click_id ?? ''}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </DirectoryTable>
      ) : null}

      {mappings ? (
        <section className="ui-filter-panel gap-3">
          <h3 className="font-semibold">Conversion mappings</h3>
          {mappingDrafts.map((draft, index) => (
            <div
              key={`mapping-${index}`}
              className="grid grid-cols-[repeat(auto-fill,minmax(10rem,1fr))] items-end gap-3"
            >
              <div className="grid gap-2">
                <Label htmlFor={`mapping-status-${index}`}>Inbound status</Label>
                <Input
                  id={`mapping-status-${index}`}
                  value={draft.inbound_status}
                  onChange={(event) => {
                    const next = [...mappingDrafts];
                    next[index] = { ...draft, inbound_status: event.target.value };
                    setMappingDrafts(next);
                  }}
                />
              </div>
              <div className="grid gap-2">
                <Label htmlFor={`mapping-goal-${index}`}>Goal name</Label>
                <Input
                  id={`mapping-goal-${index}`}
                  value={draft.goal_name}
                  onChange={(event) => {
                    const next = [...mappingDrafts];
                    next[index] = { ...draft, goal_name: event.target.value };
                    setMappingDrafts(next);
                  }}
                />
              </div>
              <div className="grid gap-2">
                <Label htmlFor={`mapping-payout-${index}`}>Payout micro</Label>
                <Input
                  id={`mapping-payout-${index}`}
                  inputMode="numeric"
                  value={draft.payout_micro}
                  onChange={(event) => {
                    const next = [...mappingDrafts];
                    next[index] = { ...draft, payout_micro: event.target.value };
                    setMappingDrafts(next);
                  }}
                />
              </div>
            </div>
          ))}
          <div className="flex flex-wrap gap-2">
            <Button
              disabled={busy}
              onClick={() =>
                setMappingDrafts((rows) => [
                  ...rows,
                  { inbound_status: '', goal_name: '', payout_micro: '' },
                ])
              }
              type="button"
              variant="outline"
            >
              Add row
            </Button>
            <Button disabled={busy} onClick={onSaveMappings} type="button">
              {savingMappings ? 'Saving...' : 'Save mappings'}
            </Button>
          </div>
          {mappingSaveSuccess ? (
            <p className="text-sm text-muted-foreground" role="status">
              Conversion mappings saved.
            </p>
          ) : null}
        </section>
      ) : null}

      {suggestions.length > 0 ? (
        <DirectoryTable>
          <TableHeader>
            <TableRow>
              <DirectoryTableHead>Placement</DirectoryTableHead>
              <DirectoryTableHead>IVT rate</DirectoryTableHead>
              <DirectoryTableHead>Reason</DirectoryTableHead>
              <DirectoryTableHead />
            </TableRow>
          </TableHeader>
          <TableBody>
            {suggestions.map((row) => (
              <TableRow key={row.placement_id}>
                <TableCell className="font-mono text-xs">{row.placement_id}</TableCell>
                <TableCell>{row.ivt_rate_label ?? row.ivt_rate ?? ''}</TableCell>
                <TableCell>{row.reason_label ?? row.suggested_action ?? ''}</TableCell>
                <TableCell>
                  <Button
                    disabled={busy}
                    onClick={() => setDraftPlacementId(row.placement_id)}
                    type="button"
                    variant="outline"
                  >
                    Use
                  </Button>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </DirectoryTable>
      ) : null}

      <div className={INLINE_FILTER_ACTION_GRID_CLASS}>
        <FilterField htmlFor="ops-placement-id" label="Placement ID to block">
          <Input
            id="ops-placement-id"
            value={draftPlacementId}
            onChange={(event) => setDraftPlacementId(event.target.value)}
          />
        </FilterField>
        <Button disabled={blocking} onClick={onBlockPlacement} type="button" variant="destructive">
          {blocking ? 'Blocking...' : 'Block placement'}
        </Button>
      </div>

      {smokeMessage ? (
        <p className="text-sm text-muted-foreground" role="status">
          {smokeMessage}
        </p>
      ) : null}
      {flowMessage ? (
        <p className="text-sm text-muted-foreground" role="status">
          Flow validation: {flowMessage}
        </p>
      ) : null}

      {actionError ? campaignPanelError(actionError, 'Campaign ops action failed') : null}
    </div>
  );
}
