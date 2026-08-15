export type TabBarTab = {
  id: string;
  label: string;
};

export type TabBarProps = {
  tabs: TabBarTab[];
  active: string;
  onChange: (id: string) => void;
};

/**
 * Horizontal tab bar for switching panel sections.
 */
export function TabBar({ tabs, active, onChange }: TabBarProps) {
  return (
    <div className="tab-bar" role="tablist">
      {tabs.map((tab) => (
        <button
          key={tab.id}
          type="button"
          role="tab"
          aria-selected={active === tab.id}
          className={`tab-bar__item${active === tab.id ? ' tab-bar__item--active' : ''}`}
          onClick={() => onChange(tab.id)}
        >
          {tab.label}
        </button>
      ))}
    </div>
  );
}
