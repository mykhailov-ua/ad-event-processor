// Human labels for platformconfig restart_required keys (pkg/platformconfig/validate.go).
const RESTART_FIELD_LABELS: Record<string, string> = {
  ingress_schema: 'Ingress schema',
  telemetry_enabled: 'Telemetry',
  edge_xdp: 'Edge XDP',
  edge_expose_click: 'Expose click endpoint',
  edge_expose_openrtb: 'Expose OpenRTB endpoint',
  profile: 'Deployment profile',
  network_interface: 'Network interface',
  stripe: 'Stripe billing',
};

export const KNOWN_RESTART_REQUIRED_KEYS = Object.keys(RESTART_FIELD_LABELS);

export function settingsFieldLabel(key: string): string {
  return RESTART_FIELD_LABELS[key] ?? key;
}

export function formatRestartRequiredLabels(keys: string[]): string[] {
  return keys.map(settingsFieldLabel);
}

const INGRESS_SCHEMA_LABELS: Record<string, string> = {
  ad_event_processor_native: 'Native JSON',
  openrtb_3: 'OpenRTB 3',
};

export function ingressSchemaLabel(schema: string): string {
  return INGRESS_SCHEMA_LABELS[schema] ?? schema;
}
