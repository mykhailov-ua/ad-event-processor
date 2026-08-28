import { useState } from 'react';
import {
  getWizardSession,
  postWizardSession,
  type WizardSession,
  type WizardStep,
} from '../../helpers/wizard_api.js';
import { ConfirmCancelledError } from '../../helpers/confirmed_api.js';
import { pushToastMessage } from '../../helpers/toast_ui.js';
import { ContextBar } from '../shell/context_bar.js';
import { PageChrome } from '../system/page_chrome.js';
import { Button } from '../system/button.js';
import { ErrorBlock } from '../system/error_block.js';
import styles from './wizard_detail.module.css';

const STEPS: WizardStep[] = [
  'traffic_source',
  'integration_template',
  'flow_skeleton',
  'budget',
];

const STEP_LABELS: Record<WizardStep, string> = {
  traffic_source: 'Traffic source',
  integration_template: 'Integration template',
  flow_skeleton: 'Flow skeleton',
  budget: 'Budget',
};

export function WizardDetail() {
  const [stepIndex, setStepIndex] = useState(0);
  const [sessionId, setSessionId] = useState('');
  const [customerId, setCustomerId] = useState('');
  const [templateKey, setTemplateKey] = useState('');
  const [payloadJson, setPayloadJson] = useState('{}');
  const [session, setSession] = useState<WizardSession | null>(null);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<unknown>(null);

  const step = STEPS[stepIndex] ?? 'traffic_source';

  const parsePayload = (): Record<string, unknown> => {
    try {
      return JSON.parse(payloadJson) as Record<string, unknown>;
    } catch {
      return {};
    }
  };

  const onCreate = async () => {
    setSaving(true);
    setError(null);
    try {
      const result = (await postWizardSession({
        action: 'create',
        customer_id: customerId || undefined,
        template_key: templateKey || undefined,
        step,
        payload: parsePayload(),
      })) as WizardSession;
      const id = result.session_id ?? '';
      setSessionId(id);
      setSession(result);
      pushToastMessage({ title: 'Session created', message: id || 'Wizard session started' });
    } catch (err) {
      if (err instanceof ConfirmCancelledError) return;
      setError(err);
    } finally {
      setSaving(false);
    }
  };

  const onUpdate = async () => {
    if (!sessionId) {
      setError(new Error('session_id required'));
      return;
    }
    setSaving(true);
    setError(null);
    try {
      const result = (await postWizardSession({
        action: 'update',
        session_id: sessionId,
        step,
        payload: parsePayload(),
      })) as WizardSession;
      setSession(result);
      pushToastMessage({ title: 'Session updated', message: `Step: ${step}` });
    } catch (err) {
      if (err instanceof ConfirmCancelledError) return;
      setError(err);
    } finally {
      setSaving(false);
    }
  };

  const onLoadSession = async () => {
    if (!sessionId) return;
    setSaving(true);
    setError(null);
    try {
      const result = await getWizardSession(sessionId);
      setSession(result);
      if (result.step) {
        const idx = STEPS.indexOf(result.step as WizardStep);
        if (idx >= 0) setStepIndex(idx);
      }
    } catch (err) {
      setError(err);
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className={styles.root}>
      <ContextBar parentLabel="Campaigns" parentTo="/campaigns" currentLabel="Wizard" />
      <PageChrome title="Campaign wizard" />
      <div className={styles.panel}>
        <div className={styles.steps}>
          {STEPS.map((s, index) => (
            <button
              key={s}
              type="button"
              className={[styles.step, index === stepIndex ? styles.stepActive : ''].join(' ')}
              onClick={() => setStepIndex(index)}
            >
              {STEP_LABELS[s]}
            </button>
          ))}
        </div>
        {error ? <ErrorBlock error={error} fallbackTitle="Wizard action failed" /> : null}
        {session?.session_id ? (
          <p className={styles.mono}>Session: {session.session_id}</p>
        ) : null}
        <label className={styles.field}>
          <span className={styles.label}>Customer ID</span>
          <input
            className={styles.input}
            value={customerId}
            onChange={(e) => setCustomerId(e.target.value)}
          />
        </label>
        <label className={styles.field}>
          <span className={styles.label}>Template key</span>
          <input
            className={styles.input}
            value={templateKey}
            onChange={(e) => setTemplateKey(e.target.value)}
          />
        </label>
        <label className={styles.field}>
          <span className={styles.label}>Session ID</span>
          <input
            className={styles.input}
            value={sessionId}
            onChange={(e) => setSessionId(e.target.value)}
          />
        </label>
        <label className={styles.field}>
          <span className={styles.label}>Step payload (JSON)</span>
          <textarea
            className={styles.input}
            rows={6}
            value={payloadJson}
            onChange={(e) => setPayloadJson(e.target.value)}
          />
        </label>
        <div className={styles.actions}>
          <Button variant="secondary" size="sm" type="button" disabled={saving} onClick={() => void onLoadSession()}>
            Load session
          </Button>
          <Button variant="secondary" size="sm" type="button" disabled={saving} onClick={() => void onCreate()}>
            {saving ? 'Working...' : 'Create session'}
          </Button>
          <Button variant="primary" size="sm" type="button" disabled={saving || !sessionId} onClick={() => void onUpdate()}>
            {saving ? 'Working...' : 'Update step'}
          </Button>
          <Button
            variant="secondary"
            size="sm"
            type="button"
            disabled={stepIndex >= STEPS.length - 1}
            onClick={() => setStepIndex((i) => Math.min(i + 1, STEPS.length - 1))}
          >
            Next step
          </Button>
        </div>
      </div>
    </div>
  );
}
