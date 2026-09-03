import { useCallback, useEffect, useState } from 'react';

import {
  blockCampaignPlacement,
  getCampaignMargin,
  getCampaignStats,
  getPlacementBlockSuggestions,
  listCampaignConversionMappings,
  listCampaignEvents,
  replaceCampaignConversionMappings,
  runCampaignSmoke,
  validateCampaignFlow,
} from '@/api/campaigns_api';
import { ApiError } from '@/api/client';
import type {
  CampaignEventListResponse,
  CampaignMargin,
  CampaignStats,
  ConversionMapping,
  ConversionMappingListResponse,
  PlacementBlockSuggestion,
} from '@/api/types';
import { ErrorBlock } from '@/shell/error_block';
import { StubBanner } from '@/shell/stub_banner';
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

function panelError(error: Error, title: string) {
  if (error instanceof ApiError && error.status === 501) {
    return <StubBanner title={`${title} unavailable`} message={error.message} />;
  }
  return <ErrorBlock title={title} message={error.message} />;
}

type MappingDraft = {
  inbound_status: string;
  goal_name: string;
  payout_micro: string;
};

function mappingToDraft(mapping: ConversionMapping): MappingDraft {
  return {
    inbound_status: mapping.inbound_status ?? '',
    goal_name: mapping.goal_name ?? '',
    payout_micro:
      mapping.payout_micro != null ? String(mapping.payout_micro) : '',
  };
}

function draftsToMappings(drafts: MappingDraft[]): ConversionMapping[] {
  return drafts.map((draft) => {
    const mapping: ConversionMapping = {
      inbound_status: draft.inbound_status.trim(),
      goal_name: draft.goal_name.trim(),
    };
    const payout = draft.payout_micro.trim();
    if (payout) {
      const parsed = Number.parseInt(payout, 10);
      if (!Number.isFinite(parsed)) {
        throw new Error('Payout micro must be an integer.');
      }
      mapping.payout_micro = parsed;
    }
    return mapping;
  });
}

export function CampaignOpsPanel({ campaignId }: { campaignId: string }) {
  const [draftPlacementId, setDraftPlacementId] = useState('');
  const [blocking, setBlocking] = useState(false);
  const [loadingKey, setLoadingKey] = useState<string | undefined>();
  const [stats, setStats] = useState<CampaignStats | undefined>();
  const [events, setEvents] = useState<CampaignEventListResponse | undefined>();
  const [margin, setMargin] = useState<CampaignMargin | undefined>();
  const [mappings, setMappings] = useState<ConversionMappingListResponse | undefined>();
  const [mappingDrafts, setMappingDrafts] = useState<MappingDraft[]>([]);
  const [suggestions, setSuggestions] = useState<PlacementBlockSuggestion[]>([]);
  const [smokeMessage, setSmokeMessage] = useState<string | undefined>();
  const [flowMessage, setFlowMessage] = useState<string | undefined>();
  const [actionError, setActionError] = useState<Error | undefined>();
  const [savingMappings, setSavingMappings] = useState(false);
  const [mappingSaveSuccess, setMappingSaveSuccess] = useState(false);

  useEffect(() => {
    setStats(undefined);
    setEvents(undefined);
    setMargin(undefined);
    setMappings(undefined);
    setMappingDrafts([]);
    setSuggestions([]);
    setSmokeMessage(undefined);
    setFlowMessage(undefined);
    setActionError(undefined);
    setMappingSaveSuccess(false);
  }, [campaignId]);

  useEffect(() => {
    if (!mappings?.mappings) {
      return;
    }
    setMappingDrafts(mappings.mappings.map(mappingToDraft));
  }, [mappings]);

  const runAction = useCallback(
    async (key: string, action: () => Promise<void>) => {
      setLoadingKey(key);
      setActionError(undefined);
      try {
        await action();
      } catch (err) {
        setActionError(err instanceof Error ? err : new Error(String(err)));
      } finally {
        setLoadingKey(undefined);
      }
    },
    [],
  );

  const onLoadStats = useCallback(() => {
    void runAction('stats', async () => {
      setStats(await getCampaignStats(campaignId));
    });
  }, [campaignId, runAction]);

  const onLoadEvents = useCallback(() => {
    void runAction('events', async () => {
      setEvents(await listCampaignEvents(campaignId, { limit: 20, offset: 0 }));
    });
  }, [campaignId, runAction]);

  const onLoadMargin = useCallback(() => {
    void runAction('margin', async () => {
      setMargin(await getCampaignMargin(campaignId));
    });
  }, [campaignId, runAction]);

  const onLoadMappings = useCallback(() => {
    void runAction('mappings', async () => {
      setMappings(await listCampaignConversionMappings(campaignId));
    });
  }, [campaignId, runAction]);

  const onLoadSuggestions = useCallback(() => {
    void runAction('suggestions', async () => {
      const result = await getPlacementBlockSuggestions(campaignId);
      setSuggestions(result.items ?? []);
    });
  }, [campaignId, runAction]);

  const onRunSmoke = useCallback(() => {
    void runAction('smoke', async () => {
      const result = await runCampaignSmoke(campaignId);
      setSmokeMessage(
        result.passed
          ? 'Smoke test passed'
          : result.failure_reason ?? 'Smoke test failed',
      );
    });
  }, [campaignId, runAction]);

  const onValidateFlow = useCallback(() => {
    void runAction('flow', async () => {
      const result = await validateCampaignFlow(campaignId);
      const status = typeof result.status === 'string' ? result.status : 'validated';
      setFlowMessage(status);
    });
  }, [campaignId, runAction]);

  const onSaveMappings = useCallback(async () => {
    setSavingMappings(true);
    setActionError(undefined);
    setMappingSaveSuccess(false);
    try {
      const body = { mappings: draftsToMappings(mappingDrafts) };
      const updated = await replaceCampaignConversionMappings(campaignId, body);
      setMappings(updated);
      setMappingSaveSuccess(true);
    } catch (err) {
      setActionError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setSavingMappings(false);
    }
  }, [campaignId, mappingDrafts]);

  const onBlockPlacement = useCallback(async () => {
    const placementId = draftPlacementId.trim();
    if (!placementId) {
      setActionError(new Error('Placement ID is required.'));
      return;
    }
    setBlocking(true);
    setActionError(undefined);
    try {
      await blockCampaignPlacement(campaignId, { placement_id: placementId });
      setDraftPlacementId('');
    } catch (err) {
      setActionError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setBlocking(false);
    }
  }, [campaignId, draftPlacementId]);

  const busy = loadingKey != null || blocking || savingMappings;

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
        <section className="ui-filter-panel gap-2 text-sm">
          <h3 className="font-semibold">Margin</h3>
          <p>
            Operator margin (micro):{' '}
            <strong>{margin.operator_margin_micro ?? ''}</strong>
          </p>
          <p>
            Advertiser spend (micro):{' '}
            <strong>{margin.advertiser_spend_micro ?? ''}</strong>
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
                      disabled={blocking}
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

      <div className="grid max-w-md grid-cols-[1fr_auto] items-end gap-4">
        <div className="grid gap-2">
          <Label htmlFor="ops-placement-id">Placement ID to block</Label>
          <Input
            id="ops-placement-id"
            value={draftPlacementId}
            onChange={(event) => setDraftPlacementId(event.target.value)}
          />
        </div>
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

      {actionError ? panelError(actionError, 'Campaign ops action failed') : null}
    </div>
  );
}
