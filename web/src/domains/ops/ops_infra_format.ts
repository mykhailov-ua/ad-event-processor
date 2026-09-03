export const OPS_DASHBOARD_METRIC_OPTIONS = [
  { value: 'ad_http_requests_total', label: 'HTTP requests (counter)' },
  { value: 'process_resident_memory_bytes', label: 'RSS memory' },
  { value: 'go_memstats_heap_inuse_bytes', label: 'Go heap in use' },
  { value: 'process_cpu_seconds_total', label: 'CPU seconds (counter)' },
  { value: 'go_goroutines', label: 'Goroutines' },
  { value: 'ad_gnet_active_connections', label: 'gnet active connections' },
  { value: 'ad_control_outbox_pending_total', label: 'Outbox pending' },
  { value: 'ad_tracker_redis_shard_healthy', label: 'Redis shard healthy' },
  { value: 'ad_worker_pool_reject_total', label: 'Worker pool rejects' },
] as const;

export function formatOpsByteCount(bytes: number | null | undefined): string {
  if (bytes == null || !Number.isFinite(bytes) || bytes < 0) {
    return '-';
  }
  if (bytes < 1024) {
    return `${bytes} B`;
  }
  if (bytes < 1024 * 1024) {
    return `${(bytes / 1024).toFixed(1)} KiB`;
  }
  if (bytes < 1024 * 1024 * 1024) {
    return `${(bytes / (1024 * 1024)).toFixed(1)} MiB`;
  }
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GiB`;
}

export function formatOpsRate(value: number | null | undefined, unit = '/s'): string {
  if (value == null || !Number.isFinite(value)) {
    return '-';
  }
  if (value >= 100) {
    return `${value.toFixed(0)}${unit}`;
  }
  if (value >= 10) {
    return `${value.toFixed(1)}${unit}`;
  }
  return `${value.toFixed(2)}${unit}`;
}
