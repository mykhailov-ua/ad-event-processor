import { OpsMetrics } from '@/domains/ops/ops_metrics';
import { useOpsMetricsPageWorkspace } from '@/domains/ops/use_ops_metrics_page_workspace';

export function OpsMetricsPage() {
  return <OpsMetrics {...useOpsMetricsPageWorkspace()} />;
}
