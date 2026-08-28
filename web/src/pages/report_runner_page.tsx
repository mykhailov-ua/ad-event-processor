import { useCallback, useMemo } from 'react';
import { useSearchParams } from 'react-router-dom';
import {
  buildReportUrl,
  type ReportFetchResponse,
} from '../helpers/report_api.js';
import { useResource } from '../helpers/use_resource.js';
import { ReportRunner } from '../ui/reports/report_runner.js';
import type { ReportFilterValues } from '../ui/reports/report_filter_panel.js';

const DEFAULT_LIMIT = '50';

export type ReportRunnerPageProps = {
  reportKey: string;
};

function readFilterValues(searchParams: URLSearchParams): ReportFilterValues {
  return {
    from: searchParams.get('from') ?? '',
    to: searchParams.get('to') ?? '',
    customerId: searchParams.get('customer_id') ?? '',
    campaignId: searchParams.get('campaign_id') ?? '',
    limit: searchParams.get('limit') ?? DEFAULT_LIMIT,
    offset: searchParams.get('offset') ?? '0',
    cursor: searchParams.get('cursor') ?? '',
  };
}

function toQueryParams(values: ReportFilterValues) {
  const limit = Number.parseInt(values.limit, 10);
  const offset = Number.parseInt(values.offset, 10);
  return {
    from: values.from || undefined,
    to: values.to || undefined,
    customer_id: values.customerId || undefined,
    campaign_id: values.campaignId || undefined,
    limit: Number.isFinite(limit) && limit > 0 ? limit : undefined,
    offset: Number.isFinite(offset) && offset >= 0 ? offset : undefined,
    cursor: values.cursor || undefined,
  };
}

export function ReportRunnerPage({ reportKey }: ReportRunnerPageProps) {
  const [searchParams, setSearchParams] = useSearchParams();
  const filterValues = useMemo(() => readFilterValues(searchParams), [searchParams]);
  const reportUrl = useMemo(
    () => buildReportUrl(reportKey, toQueryParams(filterValues)),
    [reportKey, filterValues]
  );

  const { data, loading, error, reload } = useResource<ReportFetchResponse>(reportUrl);

  const patchParams = useCallback(
    (patch: Record<string, string | null>) => {
      const next = new URLSearchParams(searchParams);
      for (const [key, value] of Object.entries(patch)) {
        if (value === null || value === '') {
          next.delete(key);
        } else {
          next.set(key, value);
        }
      }
      setSearchParams(next, { replace: true });
    },
    [searchParams, setSearchParams]
  );

  const onFilterChange = useCallback(
    (field: keyof ReportFilterValues, value: string) => {
      const paramMap: Record<keyof ReportFilterValues, string> = {
        from: 'from',
        to: 'to',
        customerId: 'customer_id',
        campaignId: 'campaign_id',
        limit: 'limit',
        offset: 'offset',
        cursor: 'cursor',
      };
      patchParams({ [paramMap[field]]: value });
    },
    [patchParams]
  );

  const onFilterApply = useCallback(() => {
    patchParams({ offset: '0', cursor: null });
  }, [patchParams]);

  const onOffsetChange = useCallback(
    (nextOffset: number) => {
      patchParams({ offset: String(nextOffset), cursor: null });
    },
    [patchParams]
  );

  return (
    <ReportRunner
      reportKey={reportKey}
      data={data}
      loading={loading}
      error={error}
      filterValues={filterValues}
      onFilterChange={onFilterChange}
      onFilterApply={onFilterApply}
      onOffsetChange={onOffsetChange}
      onReload={reload}
    />
  );
}
