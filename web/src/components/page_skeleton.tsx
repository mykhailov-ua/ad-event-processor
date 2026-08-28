export function PageSkeleton({ rows = 3 }: { rows?: number }) {
  return (
    <div className="stack stack--lg" aria-busy="true" aria-label="Loading page content">
      {}
      <div className="stack stack--sm">
        <div className="skeleton-bar" style={{ width: '45%', height: '1.5rem' }} />
        <div className="skeleton-bar" style={{ width: '65%', height: '0.875rem' }} />
      </div>

      {}
      {Array.from({ length: rows }, (_, i) => (
        <div key={i} className="settings-panel" style={{ gap: 'var(--space-sm)' }}>
          <div className="skeleton-bar" style={{ width: '30%', height: '1rem' }} />
          <div className="skeleton-bar" style={{ width: '100%' }} />
          <div className="skeleton-bar" style={{ width: '80%' }} />
          <div className="skeleton-bar" style={{ width: '90%' }} />
        </div>
      ))}
    </div>
  );
}
