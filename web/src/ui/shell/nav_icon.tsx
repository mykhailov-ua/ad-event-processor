import styles from './nav_icon.module.css';

const LABELS: Record<string, string> = {
  users: 'U',
  layers: 'A',
  'panel-left-open': '+',
  'panel-left-close': '-',
  sun: 'L',
  moon: 'D',
  'log-out': 'X',
};

export type NavIconProps = {
  name: string;
  size?: number;
};

export function NavIcon({ name, size = 16 }: NavIconProps) {
  const label = LABELS[name] ?? name.slice(0, 1).toUpperCase();
  return (
    <span className={styles.root} style={{ width: size, height: size }} aria-hidden="true">
      {label}
    </span>
  );
}
