import type { ReactNode } from 'react';

export type FilterChipItem = {
  value: string;
  label: string;
};

export type FilterToolbarProps = {
  search?: boolean;
  searchPlaceholder?: string;
  searchValue?: string;
  onSearch?: (value: string) => void;
  chips?: FilterChipItem[];
  chipSelected?: string;
  onChipSelect?: (value: string) => void;
  leading?: ReactNode;
  pagination?: ReactNode;
};

export function FilterToolbar({
  search,
  searchPlaceholder = 'Search...',
  searchValue = '',
  onSearch,
  chips,
  chipSelected = '',
  onChipSelect,
  leading,
  pagination,
}: FilterToolbarProps) {
  return (
    <div className="filter-toolbar elevation-sunken">
      {leading ? <div className="filter-toolbar__leading">{leading}</div> : null}
      {search ? (
        <div className="filter-toolbar__search">
          <input
            type="search"
            className="form-input form-input--sm filter-toolbar__search-input"
            placeholder={searchPlaceholder}
            value={searchValue}
            aria-label={searchPlaceholder}
            onChange={(e) => onSearch?.(e.target.value)}
          />
        </div>
      ) : null}
      {chips?.length ? (
        <div
          className="filter-toolbar__chips chip-row"
          role="radiogroup"
          aria-label="Filter options"
        >
          {chips.map((chip) => {
            const selected = chip.value === chipSelected;
            return (
              <button
                key={chip.value}
                type="button"
                className={`chip${selected ? ' chip--active' : ''}`}
                role="radio"
                aria-checked={selected}
                tabIndex={selected ? 0 : -1}
                onClick={() => onChipSelect?.(chip.value)}
              >
                {chip.label}
              </button>
            );
          })}
        </div>
      ) : null}
      <div className="filter-toolbar__spacer" />
      {pagination ? <div className="filter-toolbar__actions">{pagination}</div> : null}
    </div>
  );
}
