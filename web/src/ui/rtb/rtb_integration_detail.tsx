import { Fragment, useEffect, useState } from 'react';
import {
  applyRtbFloors,
  fetchRtbProfile,
  fetchRtbShadowDiff,
  RTB_DETAIL_TABS,
  validateRtbBidRequest,
  type RtbDetailTab,
  type RtbShadowDiff,
} from '../../helpers/rtb_api.js';
import { ConfirmCancelledError } from '../../helpers/confirmed_api.js';
import { pushToastMessage } from '../../helpers/toast_ui.js';
import { ContextBar } from '../shell/context_bar.js';
import { PageChrome } from '../system/page_chrome.js';
import { TabBar } from '../system/tab_bar.js';
import { Button } from '../system/button.js';
import { ErrorBlock } from '../system/error_block.js';
import { PageSkeleton } from '../system/page_skeleton.js';
import styles from './rtb_integration_detail.module.css';

export type RtbIntegrationDetailProps = {
  tab: RtbDetailTab;
  onTabChange: (tab: RtbDetailTab) => void;
};

export function RtbIntegrationDetail({ tab, onTabChange }: RtbIntegrationDetailProps) {
  const [profile, setProfile] = useState<Record<string, unknown> | null>(null);
  const [profileLoading, setProfileLoading] = useState(false);
  const [profileError, setProfileError] = useState<unknown>(null);

  const [shadow, setShadow] = useState<RtbShadowDiff | null>(null);
  const [shadowLoading, setShadowLoading] = useState(false);
  const [shadowError, setShadowError] = useState<unknown>(null);

  const [bidJson, setBidJson] = useState('{}');
  const [validateResult, setValidateResult] = useState<string | null>(null);
  const [validating, setValidating] = useState(false);
  const [validateError, setValidateError] = useState<unknown>(null);

  const [placementIds, setPlacementIds] = useState('');
  const [dryRun, setDryRun] = useState(true);
  const [floorsResult, setFloorsResult] = useState<string | null>(null);
  const [floorsError, setFloorsError] = useState<unknown>(null);
  const [applyingFloors, setApplyingFloors] = useState(false);

  useEffect(() => {
    if (tab !== 'profile') return;
    setProfileLoading(true);
    setProfileError(null);
    void fetchRtbProfile()
      .then((data) => setProfile(data))
      .catch((err) => setProfileError(err))
      .finally(() => setProfileLoading(false));
  }, [tab]);

  useEffect(() => {
    if (tab !== 'shadow') return;
    setShadowLoading(true);
    setShadowError(null);
    void fetchRtbShadowDiff()
      .then((data) => setShadow(data))
      .catch((err) => setShadowError(err))
      .finally(() => setShadowLoading(false));
  }, [tab]);

  const onValidate = async () => {
    setValidating(true);
    setValidateError(null);
    setValidateResult(null);
    try {
      const result = await validateRtbBidRequest(bidJson);
      setValidateResult(JSON.stringify(result, null, 2));
    } catch (err) {
      if (err instanceof ConfirmCancelledError) return;
      setValidateError(err);
    } finally {
      setValidating(false);
    }
  };

  const onApplyFloors = async () => {
    setApplyingFloors(true);
    setFloorsError(null);
    setFloorsResult(null);
    const ids = placementIds
      .split(/[\s,]+/)
      .map((id) => id.trim())
      .filter(Boolean);
    try {
      const result = await applyRtbFloors(ids, dryRun);
      setFloorsResult(JSON.stringify(result, null, 2));
      pushToastMessage({
        title: dryRun ? 'Dry run complete' : 'Floors applied',
        message: dryRun ? 'No changes committed' : 'Floor update requested',
      });
    } catch (err) {
      if (err instanceof ConfirmCancelledError) return;
      setFloorsError(err);
    } finally {
      setApplyingFloors(false);
    }
  };

  return (
    <div className={styles.root}>
      <ContextBar parentLabel="RTB" parentTo="/rtb/deals" currentLabel="Integration" />
      <PageChrome title="RTB integration" />
      <TabBar tabs={RTB_DETAIL_TABS} active={tab} onChange={(next) => onTabChange(next as RtbDetailTab)} />
      <div className={styles.panel} role="tabpanel">
        {tab === 'profile' ? (
          <div className={styles.panel}>
            {profileLoading ? <PageSkeleton rows={4} /> : null}
            {profileError ? <ErrorBlock error={profileError} fallbackTitle="Failed to load profile" /> : null}
            {profile ? (
              <dl className={styles.dl}>
                {Object.entries(profile).map(([key, value]) => (
                  <Fragment key={key}>
                    <dt>{key}</dt>
                    <dd>
                      {typeof value === 'object' ? JSON.stringify(value) : String(value ?? '-')}
                    </dd>
                  </Fragment>
                ))}
              </dl>
            ) : null}
          </div>
        ) : null}
        {tab === 'shadow' ? (
          <div className={styles.panel}>
            {shadowLoading ? <PageSkeleton rows={3} /> : null}
            {shadowError ? <ErrorBlock error={shadowError} fallbackTitle="Failed to load shadow diff" /> : null}
            {shadow ? (
              <div className={styles.kpiStrip}>
                <div className={styles.kpiCard}>
                  <span className={styles.kpiLabel}>Parity rate</span>
                  <span className={styles.kpiValue}>
                    {shadow.parity_rate != null ? String(shadow.parity_rate) : '-'}
                  </span>
                </div>
                <div className={styles.kpiCard}>
                  <span className={styles.kpiLabel}>Mismatch rate</span>
                  <span className={styles.kpiValue}>
                    {shadow.mismatch_rate != null ? String(shadow.mismatch_rate) : '-'}
                  </span>
                </div>
                <div className={styles.kpiCard}>
                  <span className={styles.kpiLabel}>Shadow evals</span>
                  <span className={styles.kpiValue}>
                    {shadow.shadow_evals != null ? String(shadow.shadow_evals) : '-'}
                  </span>
                </div>
                <div className={styles.kpiCard}>
                  <span className={styles.kpiLabel}>Window</span>
                  <span className={styles.kpiValue}>{shadow.window ?? '-'}</span>
                </div>
              </div>
            ) : null}
          </div>
        ) : null}
        {tab === 'validate' ? (
          <div className={styles.panel}>
            {validateError ? <ErrorBlock error={validateError} fallbackTitle="Validation failed" /> : null}
            <form
              className={styles.form}
              onSubmit={(e) => {
                e.preventDefault();
                void onValidate();
              }}
            >
              <label className={styles.field}>
                <span className={styles.label}>Bid request JSON</span>
                <textarea
                  className={styles.textarea}
                  value={bidJson}
                  onChange={(e) => setBidJson(e.target.value)}
                />
              </label>
              <div className={styles.actions}>
                <Button type="submit" variant="primary" disabled={validating}>
                  {validating ? 'Validating...' : 'Validate'}
                </Button>
              </div>
            </form>
            {validateResult ? <pre className={styles.pre}>{validateResult}</pre> : null}
          </div>
        ) : null}
        {tab === 'floors' ? (
          <div className={styles.panel}>
            {floorsError ? <ErrorBlock error={floorsError} fallbackTitle="Floors apply failed" /> : null}
            <form
              className={styles.form}
              onSubmit={(e) => {
                e.preventDefault();
                void onApplyFloors();
              }}
            >
              <label className={styles.field}>
                <span className={styles.label}>Placement IDs (comma or newline separated)</span>
                <textarea
                  className={styles.textarea}
                  value={placementIds}
                  onChange={(e) => setPlacementIds(e.target.value)}
                />
              </label>
              <label className={styles.field}>
                <span className={styles.label}>Dry run</span>
                <input type="checkbox" checked={dryRun} onChange={(e) => setDryRun(e.target.checked)} />
              </label>
              <div className={styles.actions}>
                <Button type="submit" variant="primary" disabled={applyingFloors}>
                  {applyingFloors ? 'Applying...' : dryRun ? 'Dry run' : 'Apply floors'}
                </Button>
              </div>
            </form>
            {floorsResult ? <pre className={styles.pre}>{floorsResult}</pre> : null}
          </div>
        ) : null}
      </div>
    </div>
  );
}
