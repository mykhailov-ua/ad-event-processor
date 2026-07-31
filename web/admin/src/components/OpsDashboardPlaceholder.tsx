export function OpsDashboardPlaceholder() {
  return (
    <section className="panel">
      <h2>Ops dashboard</h2>
      <p className="muted">
        Operator metrics and shard health will appear here. Until then, use the
        legacy ops dashboard or JSON API.
      </p>
      <ul>
        <li>
          <a href="/ops/dashboard">Legacy ops dashboard</a>
        </li>
        <li>
          <a href="/api/v1/ops/dashboard/summary">GET /api/v1/ops/dashboard/summary</a>
        </li>
        <li>
          <a href="/api/v1/ops/dashboard/metrics?range=24h">
            GET /api/v1/ops/dashboard/metrics
          </a>
        </li>
      </ul>
    </section>
  );
}
