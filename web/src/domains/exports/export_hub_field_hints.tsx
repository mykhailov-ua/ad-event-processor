export function ExportHubCustomerIdHint() {
  return (
    <p>
      Customer UUID for the export scope. Copy it from the Customers directory or from a campaign
      filter.
    </p>
  );
}

export function ExportHubJobIdHint() {
  return (
    <p>
      Async job identifier returned after Start export. Poll status and download when the job
      completes, or click a row in Recent exports to load it here.
    </p>
  );
}
