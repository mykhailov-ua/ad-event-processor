import { cn } from '../../lib/cn.js';
import styles from './tab_bar.module.css';

export type TabItem = {
  id: string;
  label: string;
};

export type TabBarProps = {
  tabs: TabItem[];
  active: string;
  onChange: (tabId: string) => void;
};

export function TabBar({ tabs, active, onChange }: TabBarProps) {
  return (
    <div className={styles.root} role="tablist" aria-label="Customer sections">
      {tabs.map((tab) => (
        <button
          key={tab.id}
          type="button"
          role="tab"
          className={cn(styles.tab, active === tab.id ? styles.tabActive : '')}
          aria-selected={active === tab.id}
          onClick={() => onChange(tab.id)}
        >
          {tab.label}
        </button>
      ))}
    </div>
  );
}
