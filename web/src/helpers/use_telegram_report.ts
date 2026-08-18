import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { to } from '../lib/to.js';
import * as auth from '../helpers/auth.js';
import { hasBoundCustomer } from '../helpers/buyer_session.js';
import { validateCustomerIdField, validateReportRange } from '../helpers/validators.js';
import { fetchCampaignOptions } from '../helpers/campaign_picker.js';
import { downloadReportExport, pollReportJob, submitReportExport } from '../helpers/report_api.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { createInFlightGuard } from '../lib/async_guard.js';
import {
  applyTelegramPreset,
  buildTelegramReportParams,
  createTelegramReportState,
  resolveTelegramCustomerId,
  syncTelegramReportUrl,
  type TelegramCampaignOption,
  type TelegramReportState,
} from '../helpers/telegram_report_state.js';
import { REPORT_DATE_PRESETS } from '../helpers/date_presets.js';

export function useTelegramReport(pagePath: string) {
  const [searchParams] = useSearchParams();
  const user = auth.getUser();
  const sessionScoped = hasBoundCustomer(user?.role);
  const exportGateRef = useRef(createInFlightGuard());

  const [state, setState] = useState<TelegramReportState>(() =>
    createTelegramReportState(searchParams, { sessionScoped, user })
  );
  const [campaignOptions, setCampaignOptions] = useState<TelegramCampaignOption[]>([]);
  const [exportStatus, setExportStatus] = useState<string | null>(null);
  const [exportLoading, setExportLoading] = useState(false);

  const validateBeforeLoad = useCallback((): string | null => {
    const customerId = resolveTelegramCustomerId(state, sessionScoped, user);
    if (!sessionScoped) {
      const customerErr = validateCustomerIdField(state.customerInput);
      if (customerErr) return customerErr;
    } else if (!customerId) {
      return 'customer_id required';
    }
    return validateReportRange(state.from, state.to);
  }, [state, sessionScoped, user]);

  const refreshCampaignOptions = useCallback(async () => {
    const customerId = resolveTelegramCustomerId(state, sessionScoped, user);
    if (!customerId) {
      setCampaignOptions([]);
      return;
    }
    const [opts] = await to(fetchCampaignOptions(customerId));
    let next = opts ?? [];
    if (state.campaignInput && !next.some((c) => c.id === state.campaignInput)) {
      next = [{ id: state.campaignInput, name: state.campaignInput }, ...next];
    }
    setCampaignOptions(next);
  }, [state.campaignInput, sessionScoped, user]);

  useEffect(() => {
    void refreshCampaignOptions();
  }, [refreshCampaignOptions]);

  const handleExport = useCallback(async () => {
    const customerId = resolveTelegramCustomerId(state, sessionScoped, user);
    const rangeErr = validateReportRange(state.from, state.to);
    const customerErr = sessionScoped ? null : validateCustomerIdField(state.customerInput);
    if (!customerId || rangeErr || customerErr) {
      pushToastMessage({
        title: 'Export blocked',
        message: rangeErr || customerErr || 'customer_id required',
      });
      return;
    }
    if (!exportGateRef.current.tryAcquire()) return;
    setExportLoading(true);
    setExportStatus(null);
    const result = await submitReportExport({
      customerId,
      reportKey: 'telegram',
      from: state.from,
      to: state.to,
    });
    if (!result.ok || !result.jobId) {
      setExportLoading(false);
      exportGateRef.current.release();
      setExportStatus(result.message);
      pushToastMessage({ title: 'Export failed', message: result.message });
      return;
    }
    const polled = await pollReportJob(result.jobId);
    if (polled.ok) {
      await downloadReportExport(result.jobId, 'telegram-report.csv');
      setExportStatus('Export downloaded');
      pushToastMessage({ title: 'Export ready', message: 'telegram-report.csv downloaded' });
    } else {
      setExportStatus(polled.message);
      pushToastMessage({ title: 'Export failed', message: polled.message });
    }
    setExportLoading(false);
    exportGateRef.current.release();
    syncTelegramReportUrl(pagePath, state);
  }, [state, sessionScoped, user, pagePath]);

  useEffect(() => () => exportGateRef.current.release(), []);

  const applyPreset = useCallback((preset: (typeof REPORT_DATE_PRESETS)[number]) => {
    setState((prev) => applyTelegramPreset(prev, preset));
  }, []);

  const syncUrl = useCallback(() => {
    syncTelegramReportUrl(pagePath, state);
  }, [pagePath, state]);

  const reportParams = useMemo(
    () => buildTelegramReportParams(state, sessionScoped, user),
    [state, sessionScoped, user]
  );

  return {
    state,
    setState,
    sessionScoped,
    user,
    campaignOptions,
    exportStatus,
    exportLoading,
    validateBeforeLoad,
    refreshCampaignOptions,
    handleExport,
    applyPreset,
    syncUrl,
    reportParams,
    pagePath,
  };
}
