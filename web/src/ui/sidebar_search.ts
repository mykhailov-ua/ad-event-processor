import { el } from '../lib/dom.js';
import * as auth from '../helpers/auth.js';
import { buildSearchItems, filterSearchItems, type SearchItem } from '../helpers/search_catalog.js';
import { renderIcon } from './icon.js';

export type SidebarSearchHandle = {
  destroy: () => void;
  focus: (initialQuery?: string) => void;
  close: () => void;
  isOpen: () => boolean;
};

const DROPDOWN_GAP_PX = 4;

/**
 * Anchored typeahead under the sidebar search field (no fullscreen overlay).
 * Dropdown is portaled to document.body and only mounted while open.
 */
export function installSidebarSearch(anchor: HTMLElement): SidebarSearchHandle {
  let open = false;
  let highlight = 0;
  let filtered: SearchItem[] = [];
  let rowEls: HTMLButtonElement[] = [];
  let lastFilterKey = '';
  let cachedItems: SearchItem[] | null = null;
  let cachedPermKey = '';
  let dropdown: HTMLElement | null = null;
  let repositionRaf = 0;

  const input = el('input', {
    type: 'text',
    className: 'sidebar__search-input',
    placeholder: 'Search pages…',
    'aria-label': 'Search pages',
    'aria-expanded': 'false',
    'aria-controls': 'sidebar-search-results',
    'aria-autocomplete': 'list',
    role: 'combobox',
    autocomplete: 'off',
    autocapitalize: 'off',
    autocorrect: 'off',
    spellcheck: false,
    onInput: (e: Event) => applyQuery((e.target as HTMLInputElement).value),
    onFocus: () => {
      if (input.value.trim()) show();
    },
    onKeydown: (e: Event) => {
      const ke = e as KeyboardEvent;
      if (ke.key === 'ArrowDown') {
        ke.preventDefault();
        if (!open && !input.value.trim()) return;
        if (!open) show();
        setHighlight(highlight + 1);
      } else if (ke.key === 'ArrowUp') {
        ke.preventDefault();
        setHighlight(highlight - 1);
      } else if (ke.key === 'Enter') {
        ke.preventDefault();
        if (filtered[highlight]) filtered[highlight].run();
        hide();
      } else if (ke.key === 'Escape') {
        ke.preventDefault();
        hide();
        input.blur();
      }
    },
  }) as HTMLInputElement;

  const field = el('div', { className: 'sidebar__search-field' }, input);
  const searchIcon = renderIcon('search', { size: 16, className: 'sidebar__search-icon' });

  anchor.classList.add('sidebar__search');
  anchor.replaceChildren(
    ...[searchIcon, field].filter((n): n is HTMLElement | SVGElement => n != null),
  );

  function items(): SearchItem[] {
    const permissions = auth.getUser()?.permissions ?? [];
    const permKey = permissions.join('\0');
    if (cachedItems && cachedPermKey === permKey) return cachedItems;
    cachedPermKey = permKey;
    cachedItems = buildSearchItems(() => hide());
    return cachedItems;
  }

  function invalidateCache(): void {
    cachedItems = null;
    cachedPermKey = '';
  }

  function ensureDropdown(): HTMLElement {
    if (dropdown) return dropdown;
    dropdown = el('div', {
      className: 'sidebar-search-dropdown',
      role: 'listbox',
      id: 'sidebar-search-results',
      'aria-label': 'Search results',
    });
    document.body.appendChild(dropdown);
    return dropdown;
  }

  function removeDropdown(): void {
    dropdown?.remove();
    dropdown = null;
  }

  function positionDropdown(): void {
    if (!dropdown) return;
    const rect = anchor.getBoundingClientRect();
    dropdown.style.setProperty('--sidebar-search-top', `${Math.round(rect.bottom + DROPDOWN_GAP_PX)}px`);
    dropdown.style.setProperty('--sidebar-search-left', `${Math.round(rect.left)}px`);
    dropdown.style.setProperty('--sidebar-search-width', `${Math.round(rect.width)}px`);
  }

  function scheduleReposition(): void {
    if (!open || repositionRaf) return;
    repositionRaf = requestAnimationFrame(() => {
      repositionRaf = 0;
      positionDropdown();
    });
  }

  function setHighlight(index: number): void {
    const next = Math.max(0, Math.min(index, Math.max(rowEls.length - 1, 0)));
    if (highlight === next) return;
    highlight = next;
    for (let i = 0; i < rowEls.length; i++) {
      rowEls[i].classList.toggle('sidebar-search-dropdown__item--active', i === highlight);
      rowEls[i].setAttribute('aria-selected', i === highlight ? 'true' : 'false');
    }
    rowEls[highlight]?.scrollIntoView({ block: 'nearest' });
  }

  function rebuildList(): void {
    const panel = ensureDropdown();
    const key = filtered.map((item) => item.id).join('\0');
    if (key === lastFilterKey) {
      setHighlight(highlight);
      return;
    }
    lastFilterKey = key;
    rowEls = [];
    panel.replaceChildren();

    if (filtered.length === 0) {
      panel.appendChild(el('div', { className: 'sidebar-search-dropdown__empty' }, 'No matches'));
      return;
    }

    filtered.forEach((item, i) => {
      const showHint = item.hint && item.hint !== item.label;
      const row = el('button', {
        type: 'button',
        className: 'sidebar-search-dropdown__item' + (i === highlight ? ' sidebar-search-dropdown__item--active' : ''),
        role: 'option',
        'aria-selected': i === highlight ? 'true' : 'false',
        onMouseEnter: () => setHighlight(i),
        onClick: () => {
          item.run();
          hide();
        },
      },
        el('span', { className: 'sidebar-search-dropdown__item-label' }, item.label),
        showHint
          ? el('span', { className: 'sidebar-search-dropdown__item-hint' }, item.hint)
          : null,
      ) as HTMLButtonElement;
      rowEls.push(row);
      panel.appendChild(row);
    });
  }

  function openDropdown(): void {
    if (open) {
      positionDropdown();
      return;
    }
    open = true;
    ensureDropdown();
    positionDropdown();
    anchor.classList.add('sidebar__search--open');
    input.setAttribute('aria-expanded', 'true');
  }

  function applyQuery(query: string): void {
    if (!query.trim()) {
      hide();
      return;
    }
    filtered = filterSearchItems(items(), query, hide);
    highlight = 0;
    rebuildList();
    openDropdown();
  }

  function show(): void {
    applyQuery(input.value);
  }

  function hide(): void {
    if (!open && !dropdown) return;
    open = false;
    removeDropdown();
    anchor.classList.remove('sidebar__search--open');
    input.setAttribute('aria-expanded', 'false');
    lastFilterKey = '';
    rowEls = [];
  }

  function onDocPointer(e: Event): void {
    if (!open) return;
    const t = e.target;
    if (t instanceof Node && (anchor.contains(t) || dropdown?.contains(t))) return;
    hide();
  }

  function onRoute(): void {
    invalidateCache();
    hide();
  }

  document.addEventListener('pointerdown', onDocPointer, true);
  window.addEventListener('routechange', onRoute);
  window.addEventListener('popstate', onRoute);
  window.addEventListener('scroll', scheduleReposition, { capture: true, passive: true });
  window.addEventListener('resize', scheduleReposition, { passive: true });

  return {
    isOpen: () => open,
    focus(initialQuery = '') {
      input.value = initialQuery;
      input.focus();
      if (initialQuery.trim()) show();
      else if (initialQuery) {
        input.setSelectionRange(initialQuery.length, initialQuery.length);
      }
    },
    close: hide,
    destroy() {
      hide();
      if (repositionRaf) cancelAnimationFrame(repositionRaf);
      document.removeEventListener('pointerdown', onDocPointer, true);
      window.removeEventListener('routechange', onRoute);
      window.removeEventListener('popstate', onRoute);
      window.removeEventListener('scroll', scheduleReposition, { capture: true });
      window.removeEventListener('resize', scheduleReposition);
    },
  };
}
