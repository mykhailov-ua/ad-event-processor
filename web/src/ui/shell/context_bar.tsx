import { Link } from 'react-router-dom';
import styles from './context_bar.module.css';

export type ContextBarProps = {
  parentLabel: string;
  parentTo: string;
  currentLabel: string;
};

export function ContextBar({ parentLabel, parentTo, currentLabel }: ContextBarProps) {
  return (
    <nav className={styles.root} aria-label="Breadcrumb">
      <Link to={parentTo} className={styles.link}>
        {parentLabel}
      </Link>
      <span className={styles.sep} aria-hidden="true">
        /
      </span>
      <span className={styles.current}>{currentLabel}</span>
    </nav>
  );
}
