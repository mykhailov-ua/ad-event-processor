const CAMPAIGN_JSON_FIELD_LABELS: Record<string, string> = {
  allowed_actions: 'Allowed actions',
  campaign_id: 'Campaign ID',
  cards: 'Fraud signals',
  code: 'Code',
  complete: 'Complete',
  completion_pct: 'Completion',
  conflict_warning: 'Geo conflict',
  context_links: 'Context links',
  count_label: 'Count',
  disclaimer: 'Disclaimer',
  excluded_label: 'Excluded countries',
  expanded: 'Country rows',
  field_errors: 'Field errors',
  fields: 'Fields',
  geo_summary: 'Geo summary',
  id: 'ID',
  included_label: 'Included countries',
  issue_count: 'Issues',
  issue_tone: 'Issue tone',
  kind: 'Kind',
  label: 'Country',
  order: 'Order',
  overall_status: 'Status',
  overall_status_label: 'Status',
  report_href: 'Report link',
  report_key: 'Report',
  rows: 'Rows',
  schedule_preview: 'Schedule preview',
  sections: 'Editor sections',
  severity: 'Severity',
  signal_key: 'Signal',
  title: 'Title',
  title_label: 'Signal',
  truncated: 'List truncated',
  valid: 'Valid',
  visible: 'Visible',
  warnings: 'Warnings',
};

function titleCaseFromSlug(slug: string): string {
  return slug
    .split('_')
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(' ');
}

export function formatCampaignJsonKey(key: string): string {
  return CAMPAIGN_JSON_FIELD_LABELS[key] ?? titleCaseFromSlug(key);
}
