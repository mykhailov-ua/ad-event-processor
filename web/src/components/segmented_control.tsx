export type SegmentedControlItem = {
  value: string;
  label: string;
};

export type SegmentedControlProps = {
  items: SegmentedControlItem[];
  selected: string;
  onChange: (value: string) => void;
};

export function SegmentedControl({ items, selected, onChange }: SegmentedControlProps) {
  return (
    <div className="segmented-control" role="tablist">
      {items.map((item) => {
        const isSel = item.value === selected;
        return (
          <button
            key={item.value}
            type="button"
            role="tab"
            aria-selected={isSel}
            tabIndex={isSel ? 0 : -1}
            className={`segmented-control__btn${isSel ? ' segmented-control__btn--active' : ''}`}
            onClick={() => onChange(item.value)}
          >
            {item.label}
          </button>
        );
      })}
    </div>
  );
}
