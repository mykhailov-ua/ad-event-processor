export type BootstrapBannerProps = {
  bootstrapComplete?: boolean;
};

export function BootstrapBanner({ bootstrapComplete = true }: BootstrapBannerProps) {
  if (bootstrapComplete) return null;
  return (
    <div className="stub-banner mb-4">
      <span>Platform bootstrap is not complete. </span>
      <a href="/bootstrap" style={{ color: 'var(--accent)' }}>
        Run bootstrap
      </a>
    </div>
  );
}
