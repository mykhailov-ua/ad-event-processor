import styles from './stub_banner.module.css';

export type StubBannerProps = {
  title: string;
  message: string;
};

export function StubBanner({ title, message }: StubBannerProps) {
  return (
    <div className={styles.root} role="status">
      <strong>{title}</strong>
      <p>{message}</p>
    </div>
  );
}
