import { useEffect, useState } from 'react';
import {
  DEFAULT_FLOW_PATHS,
  fetchLanders,
  fetchOffers,
  updateFlow,
  validateCampaignFlow,
  type Flow,
  type FlowBuilderTab,
  type FlowPathRef,
  type Lander,
  type Offer,
} from '../../helpers/flows_api.js';
import { ConfirmCancelledError } from '../../helpers/confirmed_api.js';
import { pushToastMessage } from '../../helpers/toast_ui.js';
import { ContextBar } from '../shell/context_bar.js';
import { PageChrome } from '../system/page_chrome.js';
import { TabBar } from '../system/tab_bar.js';
import { Button } from '../system/button.js';
import { ErrorBlock } from '../system/error_block.js';
import { StubBanner } from '../system/stub_banner.js';
import { FLOW_BUILDER_TABS } from '../../helpers/flows_api.js';
import styles from './flow_builder_detail.module.css';

export type FlowBuilderDetailProps = {
  flowId: string;
  flow: Flow;
  tab: FlowBuilderTab;
  campaignId?: string;
  onTabChange: (tab: FlowBuilderTab) => void;
  onReload: () => void;
};

type EditablePath = {
  weight: string;
  lander_id: string;
  offer_id: string;
};

function parsePaths(raw: Flow['paths']): FlowPathRef[] {
  if (raw == null) return [...DEFAULT_FLOW_PATHS];
  if (typeof raw === 'string') {
    try {
      const parsed = JSON.parse(raw) as FlowPathRef[];
      return Array.isArray(parsed) ? parsed : [...DEFAULT_FLOW_PATHS];
    } catch {
      return [...DEFAULT_FLOW_PATHS];
    }
  }
  return Array.isArray(raw) && raw.length > 0 ? raw : [...DEFAULT_FLOW_PATHS];
}

function toEditable(paths: FlowPathRef[]): EditablePath[] {
  return paths.map((path) => ({
    weight: path.weight != null ? String(path.weight) : '100',
    lander_id: path.landers?.[0]?.lander_id ?? '',
    offer_id: path.offers?.[0]?.offer_id ?? '',
  }));
}

function toFlowPaths(editable: EditablePath[]): FlowPathRef[] {
  return editable.map((path) => ({
    weight: Number.parseInt(path.weight, 10) || 0,
    landers: path.lander_id ? [{ lander_id: path.lander_id, weight: 100 }] : [],
    offers: path.offer_id ? [{ offer_id: path.offer_id, weight: 100 }] : [],
  }));
}

export function FlowBuilderDetail({
  flowId,
  flow,
  tab,
  campaignId: campaignIdProp = '',
  onTabChange,
  onReload,
}: FlowBuilderDetailProps) {
  const [paths, setPaths] = useState<EditablePath[]>(() => toEditable(parsePaths(flow.paths)));
  const [name, setName] = useState(flow.name ?? '');
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState<unknown>(null);
  const [landers, setLanders] = useState<Lander[]>([]);
  const [offers, setOffers] = useState<Offer[]>([]);
  const [catalogError, setCatalogError] = useState<unknown>(null);
  const [validateResult, setValidateResult] = useState<string | null>(null);
  const [validating, setValidating] = useState(false);
  const [campaignIdDraft, setCampaignIdDraft] = useState(campaignIdProp);

  useEffect(() => {
    setCampaignIdDraft(campaignIdProp);
  }, [campaignIdProp]);

  useEffect(() => {
    setPaths(toEditable(parsePaths(flow.paths)));
    setName(flow.name ?? '');
  }, [flow]);

  useEffect(() => {
    if (tab !== 'catalog') return;
    setCatalogError(null);
    void Promise.all([fetchLanders(), fetchOffers()])
      .then(([landerList, offerList]) => {
        setLanders(landerList);
        setOffers(offerList);
      })
      .catch((err) => setCatalogError(err));
  }, [tab]);

  const onSave = async () => {
    setSaving(true);
    setSaveError(null);
    try {
      await updateFlow(flowId, { name, paths: toFlowPaths(paths) });
      pushToastMessage({ title: 'Saved', message: 'Flow updated' });
      onReload();
    } catch (err) {
      if (err instanceof ConfirmCancelledError) return;
      setSaveError(err);
    } finally {
      setSaving(false);
    }
  };

  const onValidate = async () => {
    const campaignId = campaignIdDraft.trim();
    if (!campaignId) {
      setValidateResult('Campaign ID is required for server-side flow validation.');
      return;
    }
    setValidating(true);
    setValidateResult(null);
    try {
      const result = await validateCampaignFlow(campaignId, toFlowPaths(paths));
      setValidateResult(JSON.stringify(result, null, 2));
    } catch (err) {
      if (err instanceof ConfirmCancelledError) return;
      setValidateResult(String(err));
    } finally {
      setValidating(false);
    }
  };

  return (
    <div className={styles.root}>
      <ContextBar parentLabel="Flows" parentTo="/campaigns/flows" currentLabel={flow.name ?? flowId} />
      <PageChrome title={flow.name ?? 'Flow builder'} />
      <TabBar
        tabs={FLOW_BUILDER_TABS}
        active={tab}
        onChange={(next) => onTabChange(next as FlowBuilderTab)}
      />
      <div className={styles.panel} role="tabpanel">
        {tab === 'graph' ? (
          <div className={styles.panel}>
            {saveError ? <ErrorBlock error={saveError} fallbackTitle="Save failed" /> : null}
            <label className={styles.field}>
              <span className={styles.label}>Flow name</span>
              <input className={styles.input} value={name} onChange={(e) => setName(e.target.value)} />
            </label>
            {paths.map((path, index) => (
              <div key={`path-${index}`} className={styles.pathCard}>
                <span className={styles.label}>{`Path ${index + 1}`}</span>
                <div className={styles.pathGrid}>
                  <label className={styles.field}>
                    <span className={styles.label}>Weight</span>
                    <input
                      className={styles.input}
                      inputMode="numeric"
                      value={path.weight}
                      onChange={(e) => {
                        const next = [...paths];
                        next[index] = { ...next[index], weight: e.target.value };
                        setPaths(next);
                      }}
                    />
                  </label>
                  <label className={styles.field}>
                    <span className={styles.label}>Lander ID</span>
                    <input
                      className={styles.input}
                      value={path.lander_id}
                      onChange={(e) => {
                        const next = [...paths];
                        next[index] = { ...next[index], lander_id: e.target.value };
                        setPaths(next);
                      }}
                    />
                  </label>
                  <label className={styles.field}>
                    <span className={styles.label}>Offer ID</span>
                    <input
                      className={styles.input}
                      value={path.offer_id}
                      onChange={(e) => {
                        const next = [...paths];
                        next[index] = { ...next[index], offer_id: e.target.value };
                        setPaths(next);
                      }}
                    />
                  </label>
                </div>
              </div>
            ))}
            <div className={styles.actions}>
              <Button
                variant="secondary"
               
                type="button"
                onClick={() => setPaths([...paths, { weight: '0', lander_id: '', offer_id: '' }])}
              >
                Add path
              </Button>
              <Button variant="primary" type="button" disabled={saving} onClick={() => void onSave()}>
                {saving ? 'Saving...' : 'Save flow'}
              </Button>
            </div>
          </div>
        ) : null}
        {tab === 'catalog' ? (
          <div className={styles.panel}>
            {catalogError ? <ErrorBlock error={catalogError} fallbackTitle="Failed to load catalog" /> : null}
            <span className={styles.label}>Landers</span>
            <div className={styles.table}>
              <div className={styles.tableHead}>
                <span>Name</span>
                <span>ID</span>
                <span>URL</span>
              </div>
              {landers.map((lander) => (
                <div key={lander.id} className={styles.tableRow}>
                  <span>{lander.name ?? '-'}</span>
                  <span className={styles.mono}>{lander.id ?? '-'}</span>
                  <span className={styles.mono}>{lander.url ?? lander.hosted_url ?? '-'}</span>
                </div>
              ))}
            </div>
            <span className={styles.label}>Offers</span>
            <div className={styles.table}>
              <div className={styles.tableHead}>
                <span>Name</span>
                <span>ID</span>
                <span>URL</span>
              </div>
              {offers.map((offer) => (
                <div key={offer.id} className={styles.tableRow}>
                  <span>{offer.name ?? '-'}</span>
                  <span className={styles.mono}>{offer.id ?? '-'}</span>
                  <span className={styles.mono}>{offer.url ?? '-'}</span>
                </div>
              ))}
            </div>
          </div>
        ) : null}
        {tab === 'validate' ? (
          <div className={styles.panel}>
            {!campaignIdProp ? (
              <StubBanner
                title="Campaign context"
                message="Validation runs against a campaign flow endpoint. Pass ?campaign_id= on the URL or enter a campaign ID below."
              />
            ) : null}
            <label className={styles.field}>
              <span className={styles.label}>Campaign ID</span>
              <input
                className={styles.input}
                value={campaignIdDraft}
                onChange={(e) => setCampaignIdDraft(e.target.value)}
                placeholder="Campaign UUID"
              />
            </label>
            <div className={styles.actions}>
              <Button
                variant="secondary"
               
                type="button"
                disabled={validating || !campaignIdDraft.trim()}
                onClick={() => void onValidate()}
              >
                {validating ? 'Validating...' : 'Validate paths'}
              </Button>
            </div>
            {validateResult ? <pre className={styles.pre}>{validateResult}</pre> : null}
          </div>
        ) : null}
      </div>
    </div>
  );
}
