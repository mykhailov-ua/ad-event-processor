import type { ReactNode } from 'react';
import { TabBar } from '../system/tab_bar.js';
import { PageChrome } from '../system/page_chrome.js';
import styles from './flows_hub.module.css';

export type FlowsTab = 'landers' | 'offers' | 'flows';

export type FlowsHubProps = {
  activeTab: FlowsTab;
  onTabChange: (tab: FlowsTab) => void;
  children: ReactNode;
};

const TABS = [
  { id: 'landers', label: 'Landers' },
  { id: 'offers', label: 'Offers' },
  { id: 'flows', label: 'Flows' },
];

export function FlowsHub({ activeTab, onTabChange, children }: FlowsHubProps) {
  return (
    <div className={styles.root}>
      <PageChrome title="Campaign flows" />
      <TabBar
        tabs={TABS}
        active={activeTab}
        onChange={(tabId) => onTabChange(tabId as FlowsTab)}
      />
      <div className={styles.content}>{children}</div>
    </div>
  );
}
