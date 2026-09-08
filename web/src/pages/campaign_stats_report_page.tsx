import { CampaignStatsReportDirectory } from '@/domains/reports/campaign_stats_report_directory';
import { useCampaignStatsReportWorkspace } from '@/domains/reports/use_campaign_stats_report_workspace';

export function CampaignStatsReportPage() {
  return <CampaignStatsReportDirectory {...useCampaignStatsReportWorkspace()} />;
}
