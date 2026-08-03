import { el } from '../lib/dom.js';
import { navigate } from '../lib/router.js';
import { renderIcon } from './icon.js';
import * as auth from '../helpers/auth.js';
import * as storage from '../helpers/storage.js';
import { NAV_GROUPS, navLinkVisible } from '../helpers/nav_config.js';
import { isCustomerUuid, shortCustomerId } from '../helpers/customer_context.js';

/**
 * Install the global command palette with Ctrl/Cmd+K shortcut.
 *
 * @returns {{ destroy: () => void, open: (initialQuery?: string) => void }}
 */
export function installCommandPalette() {
  let overlay = null;
  let input = null;
  let listEl = null;
  let highlight = 0;
  let filtered = [];
  /** @type {HTMLButtonElement[]} */
  let rowEls = [];
  let lastFilterKey = '';
  /** @type {{ id: string, label: string, hint?: string, run: () => void }[]|null} */
  let cachedItems = null;
  let cachedPermKey = '';

  function buildItems() {
    const user = auth.getUser();
    const permissions = user?.permissions ?? [];
    const permKey = permissions.join('\0');
    if (cachedItems && cachedPermKey === permKey) return cachedItems;
    /** @type {{ id: string, label: string, hint?: string, run: () => void }[]} */
    const items = [];

    for (const group of NAV_GROUPS) {
      for (const link of group.links) {
        if (!navLinkVisible(permissions, link)) continue;
        items.push({
          id: `nav-${link.to}`,
          label: link.label,
          hint: group.title,
          run: () => {
            close();
            navigate(link.to);
          },
        });
      }
    }

    for (const id of storage.getRecentCustomerIds()) {
      if (navLinkVisible(permissions, { perm: 'customers:read' })) {
        items.push({
          id: `customer-${id}`,
          label: shortCustomerId(id, 12),
          hint: 'Recent customer',
          run: () => {
            close();
            navigate(`/customers/${id}`);
          },
        });
      }
      if (navLinkVisible(permissions, { perm: 'customers:read', altPerm: 'billing:read' })) {
        items.push({
          id: `billing-${id}`,
          label: `Billing · ${shortCustomerId(id, 12)}`,
          hint: 'Customer billing',
          run: () => {
            close();
            navigate(`/billing?customer_id=${encodeURIComponent(id)}`);
          },
        });
      }
      if (navLinkVisible(permissions, { perm: 'campaigns:read', altPerm: 'campaigns:read:masked' })) {
        items.push({
          id: `campaigns-${id}`,
          label: `Campaigns · ${shortCustomerId(id, 12)}`,
          hint: 'Customer campaigns',
          run: () => {
            close();
            navigate(`/campaigns?customer_id=${encodeURIComponent(id)}`);
          },
        });
      }
    }

    if (navLinkVisible(permissions, { perm: 'shards:read' })) {
      items.push({
        id: 'ops-outbox',
        label: 'Open ops outbox',
        hint: 'Operations',
        run: () => {
          close();
          navigate('/ops');
        },
      });
    }
    if (navLinkVisible(permissions, { perm: 'campaigns:write' })) {
      items.push({
        id: 'action-pause-hint',
        label: 'Pause campaign (open detail first)',
        hint: 'Action',
        run: () => {
          close();
          navigate('/campaigns');
        },
      });
    }

    cachedPermKey = permKey;
    cachedItems = items;
    return items;
  }

  function filterItems(query) {
    const q = query.trim().toLowerCase();
    const all = buildItems();
    if (!q) return all.slice(0, 12);
    const matches = all.filter((item) =>
      item.label.toLowerCase().includes(q)
      || (item.hint?.toLowerCase().includes(q))
      || item.id.toLowerCase().includes(q),
    );
    if (isCustomerUuid(q)) {
      const id = query.trim();
      matches.unshift({
        id: `goto-customer-${id}`,
        label: `Open customer ${shortCustomerId(id, 12)}`,
        hint: 'UUID match',
        run: () => {
          close();
          navigate(`/customers/${id}`);
        },
      });
    }
    return matches.slice(0, 12);
  }

  /**
   * @param {Array<{ id: string }>} items
   * @returns {string}
   */
  function filterKey(items) {
    return items.map((item) => item.id).join('\0');
  }

  function setHighlight(index) {
    const next = Math.max(0, Math.min(index, Math.max(rowEls.length - 1, 0)));
    if (highlight === next) return;
    highlight = next;
    updateHighlight();
  }

  function updateHighlight() {
    for (let i = 0; i < rowEls.length; i++) {
      rowEls[i].classList.toggle('cmd-palette__item--active', i === highlight);
    }
    const active = rowEls[highlight];
    if (active) active.scrollIntoView({ block: 'nearest' });
  }

  function rebuildList() {
    if (!listEl) return;
    const key = filterKey(filtered);
    if (key === lastFilterKey) {
      updateHighlight();
      return;
    }
    lastFilterKey = key;
    rowEls = [];
    listEl.replaceChildren();

    if (filtered.length === 0) {
      listEl.appendChild(el('div', { className: 'cmd-palette__empty' }, 'No matches'));
      return;
    }

    filtered.forEach((item, i) => {
      const showHint = item.hint && item.hint !== item.label;
      const row = el('button', {
        type: 'button',
        className: 'cmd-palette__item' + (i === highlight ? ' cmd-palette__item--active' : ''),
        onMouseEnter: () => setHighlight(i),
        onClick: () => item.run(),
      },
        el('span', { className: 'cmd-palette__item-label' }, item.label),
        showHint
          ? el('span', { className: 'cmd-palette__item-hint' }, item.hint)
          : null,
      );
      rowEls.push(row);
      listEl.appendChild(row);
    });
  }

  function applyQuery(query) {
    filtered = filterItems(query);
    highlight = 0;
    rebuildList();
  }

  /**
   * @param {string} [initialQuery]
   */
  function open(initialQuery = '') {
    if (overlay) {
      if (input) {
        input.value = initialQuery;
        applyQuery(initialQuery);
        input.focus();
        if (initialQuery) {
          input.setSelectionRange(initialQuery.length, initialQuery.length);
        }
      }
      return;
    }

    highlight = 0;
    filtered = filterItems(initialQuery);
    lastFilterKey = '';

    overlay = el('div', {
      className: 'cmd-palette-overlay',
      role: 'presentation',
      onClick: () => close(),
    });

    const dialog = el('div', {
      className: 'cmd-palette',
      role: 'dialog',
      'aria-modal': 'true',
      'aria-label': 'Search',
      onClick: (e) => e.stopPropagation(),
    });

    const inputWrap = el('div', { className: 'cmd-palette__input-wrap' },
      renderIcon('search', { size: 18, className: 'cmd-palette__search-icon' }),
    );

    input = el('input', {
      type: 'search',
      className: 'cmd-palette__input',
      placeholder: 'Search pages, customers, billing…',
      autocomplete: 'off',
      value: initialQuery,
      onInput: (e) => applyQuery(e.target.value),
      onKeydown: (e) => {
        if (e.key === 'ArrowDown') {
          e.preventDefault();
          setHighlight(highlight + 1);
        } else if (e.key === 'ArrowUp') {
          e.preventDefault();
          setHighlight(highlight - 1);
        } else if (e.key === 'Enter') {
          e.preventDefault();
          if (filtered[highlight]) filtered[highlight].run();
        } else if (e.key === 'Escape') {
          e.preventDefault();
          close();
        }
      },
    });

    inputWrap.appendChild(input);

    listEl = el('div', { className: 'cmd-palette__list', role: 'listbox' });
    rebuildList();

    dialog.appendChild(inputWrap);
    dialog.appendChild(listEl);
    overlay.appendChild(dialog);
    document.body.appendChild(overlay);
    input.focus();
    if (initialQuery) {
      input.setSelectionRange(initialQuery.length, initialQuery.length);
    }
  }

  function close() {
    if (overlay) overlay.remove();
    overlay = null;
    input = null;
    listEl = null;
    rowEls = [];
    lastFilterKey = '';
  }

  function onGlobalKey(e) {
    if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
      e.preventDefault();
      if (overlay) close();
      else open();
      return;
    }
    if (e.key === 'Escape' && overlay) {
      e.preventDefault();
      close();
    }
  }

  document.addEventListener('keydown', onGlobalKey);

  return {
    open,
    destroy: () => {
      close();
      document.removeEventListener('keydown', onGlobalKey);
    },
  };
}
