export function NotFoundPage() {
  return (
    <div className="error-page">
      <div className="error-page__code">404</div>
      <div className="error-page__title">Page not found</div>
      <div className="error-page__desc text-muted mb-4">
        The requested route does not exist or is not implemented yet.
      </div>
      <a href="/" className="btn btn--secondary btn--sm">
        Home
      </a>
    </div>
  );
}
