export type StubBannerProps = {
  message?: string;
  linkTo?: string;
  linkLabel?: string;
};

/**
 * 501 stub endpoint banner with optional link.
 */
export function StubBanner({
  message = 'Endpoint not implemented (501).',
  linkTo,
  linkLabel = 'Open placements report',
}: StubBannerProps) {
  return (
    <div className="stub-banner">
      <p className="stub-banner__message">{message}</p>
      {linkTo ? (
        <a href={linkTo} className="stub-banner__link">{linkLabel}</a>
      ) : null}
    </div>
  );
}
