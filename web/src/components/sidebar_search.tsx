import {
  forwardRef,
  useCallback,
  useEffect,
  useImperativeHandle,
  useMemo,
  useRef,
  useState,
} from 'react';
import { createPortal } from 'react-dom';
import { useLocation } from 'react-router-dom';
import * as auth from '../helpers/auth.js';
import { buildSearchItems, filterSearchItems, type SearchItem } from '../helpers/search_catalog.js';
import { Icon } from './icon.js';

const DROPDOWN_GAP_PX = 4;

export type SidebarSearchHandle = {
  focus: (initialQuery?: string) => void;
  close: () => void;
};

/**
 * Sidebar typeahead search with portaled dropdown.
 */
export const SidebarSearch = forwardRef<SidebarSearchHandle>(function SidebarSearch(_, ref) {
  const location = useLocation();
  const anchorRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  const [query, setQuery] = useState('');
  const [open, setOpen] = useState(false);
  const [highlight, setHighlight] = useState(0);
  const [dropdownStyle, setDropdownStyle] = useState<Record<string, string>>({});

  const hide = useCallback(() => {
    setOpen(false);
    setHighlight(0);
  }, []);

  const items = useMemo(() => {
    const permissions = auth.getUser()?.permissions ?? [];
    void permissions;
    return buildSearchItems(() => hide());
  }, [hide, location.pathname]);

  const filtered = useMemo(() => {
    if (!query.trim()) return [];
    return filterSearchItems(items, query, hide);
  }, [items, query, hide]);

  const positionDropdown = useCallback(() => {
    const anchor = anchorRef.current;
    if (!anchor) return;
    const rect = anchor.getBoundingClientRect();
    setDropdownStyle({
      '--sidebar-search-top': `${Math.round(rect.bottom + DROPDOWN_GAP_PX)}px`,
      '--sidebar-search-left': `${Math.round(rect.left)}px`,
      '--sidebar-search-width': `${Math.round(rect.width)}px`,
    });
  }, []);

  const showDropdown = useCallback(() => {
    positionDropdown();
    setOpen(true);
  }, [positionDropdown]);

  useEffect(() => {
    hide();
  }, [location.pathname, hide]);

  useEffect(() => {
    if (!open) return undefined;
    const onPointer = (e: PointerEvent) => {
      const t = e.target;
      const anchor = anchorRef.current;
      if (!(t instanceof Node)) return;
      if (anchor?.contains(t)) return;
      const panel = document.getElementById('sidebar-search-results');
      if (panel?.contains(t)) return;
      hide();
    };
    const onScroll = () => positionDropdown();
    document.addEventListener('pointerdown', onPointer, true);
    window.addEventListener('scroll', onScroll, { capture: true, passive: true });
    window.addEventListener('resize', onScroll, { passive: true });
    return () => {
      document.removeEventListener('pointerdown', onPointer, true);
      window.removeEventListener('scroll', onScroll, { capture: true });
      window.removeEventListener('resize', onScroll);
    };
  }, [open, hide, positionDropdown]);

  useImperativeHandle(ref, () => ({
    focus(initialQuery = '') {
      const input = inputRef.current;
      if (!input) return;
      setQuery(initialQuery);
      input.focus();
      if (initialQuery.trim()) showDropdown();
      else if (initialQuery) {
        input.setSelectionRange(initialQuery.length, initialQuery.length);
      }
    },
    close: hide,
  }), [hide, showDropdown]);

  const applyQuery = (value: string) => {
    setQuery(value);
    if (!value.trim()) {
      hide();
      return;
    }
    setHighlight(0);
    showDropdown();
  };

  const selectItem = (item: SearchItem) => {
    item.run();
    hide();
    setQuery('');
  };

  const onKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'ArrowDown') {
      e.preventDefault();
      if (!open && !query.trim()) return;
      if (!open) showDropdown();
      setHighlight((h) => Math.min(h + 1, Math.max(filtered.length - 1, 0)));
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      setHighlight((h) => Math.max(h - 1, 0));
    } else if (e.key === 'Enter') {
      e.preventDefault();
      const item = filtered[highlight];
      if (item) selectItem(item);
    } else if (e.key === 'Escape') {
      e.preventDefault();
      hide();
      inputRef.current?.blur();
    }
  };

  const dropdown = open && query.trim() ? createPortal(
    <div
      className="sidebar-search-dropdown"
      role="listbox"
      id="sidebar-search-results"
      aria-label="Search results"
      style={dropdownStyle as React.CSSProperties}
    >
      {filtered.length === 0 ? (
        <div className="sidebar-search-dropdown__empty">No matches</div>
      ) : (
        filtered.map((item, i) => {
          const showHint = item.hint && item.hint !== item.label;
          return (
            <button
              key={item.id}
              type="button"
              className={`sidebar-search-dropdown__item${i === highlight ? ' sidebar-search-dropdown__item--active' : ''}`}
              role="option"
              aria-selected={i === highlight}
              onMouseEnter={() => setHighlight(i)}
              onClick={() => selectItem(item)}
            >
              <span className="sidebar-search-dropdown__item-label">{item.label}</span>
              {showHint ? (
                <span className="sidebar-search-dropdown__item-hint">{item.hint}</span>
              ) : null}
            </button>
          );
        })
      )}
    </div>,
    document.body,
  ) : null;

  return (
    <>
      <div
        ref={anchorRef}
        className={`sidebar__search${open ? ' sidebar__search--open' : ''}`}
      >
        <Icon name="search" size={16} className="sidebar__search-icon" />
        <div className="sidebar__search-field">
          <input
            ref={inputRef}
            type="text"
            className="sidebar__search-input"
            placeholder="Search pages…"
            aria-label="Search pages"
            aria-expanded={open}
            aria-controls="sidebar-search-results"
            aria-autocomplete="list"
            role="combobox"
            autoComplete="off"
            autoCapitalize="off"
            autoCorrect="off"
            spellCheck={false}
            value={query}
            onChange={(e) => applyQuery(e.target.value)}
            onFocus={() => {
              if (query.trim()) showDropdown();
            }}
            onKeyDown={onKeyDown}
          />
        </div>
      </div>
      {dropdown}
    </>
  );
});
