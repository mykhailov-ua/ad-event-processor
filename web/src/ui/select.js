import { el, appendChildren } from '../lib/dom.js';

/**
 * @typedef {{ value: string, label: string }} SelectOption
 */

/**
 * @param {string[] | SelectOption[]} options
 * @returns {SelectOption[]}
 */
function normalizeOptions(options) {
  return options.map((o) =>
    typeof o === 'string' ? { value: o, label: o } : o,
  );
}

/**
 * @param {{
 *   value: string,
 *   options: string[] | SelectOption[],
 *   onChange: (value: string) => void,
 *   disabled?: boolean,
 *   id?: string,
 *   className?: string,
 *   style?: Record<string, string>,
 *   'aria-label'?: string,
 * }} opts
 */
export function renderSelect(opts) {
  const options = normalizeOptions(opts.options);
  const selected = options.find((o) => o.value === opts.value) ?? options[0];
  const listId = opts.id ? `${opts.id}-list` : `select-list-${Math.random().toString(36).slice(2, 9)}`;
  let open = false;
  let highlight = options.findIndex((o) => o.value === opts.value);
  if (highlight < 0) highlight = 0;

  const root = el('div', {
    className: [
      'select',
      opts.disabled ? 'select--disabled' : '',
      opts.className ?? '',
    ].filter(Boolean).join(' '),
    style: opts.style,
  });

  const trigger = el('button', {
    type: 'button',
    className: 'select__trigger',
    id: opts.id,
    disabled: opts.disabled,
    'aria-label': opts['aria-label'],
    'aria-haspopup': 'listbox',
    'aria-expanded': 'false',
    'aria-controls': listId,
  });

  const valueEl = el('span', { className: 'select__value' }, selected?.label ?? '');
  const chevron = el('span', { className: 'select__chevron', 'aria-hidden': 'true' });

  trigger.appendChild(valueEl);
  trigger.appendChild(chevron);

  const list = el('div', {
    className: 'select__list',
    id: listId,
    role: 'listbox',
    hidden: true,
  });

  const optionNodes = options.map((o, i) =>
    el('div', {
      className: [
        'select__option',
        o.value === opts.value ? 'select__option--selected' : '',
      ].join(' '),
      role: 'option',
      'aria-selected': o.value === opts.value ? 'true' : 'false',
      id: `${listId}-opt-${i}`,
      onMouseDown: (e) => {
        e.preventDefault();
        pick(o.value);
      },
    }, o.label),
  );

  appendChildren(list, optionNodes);

  function setOpen(next) {
    if (opts.disabled) return;
    open = next;
    root.classList.toggle('select--open', open);
    trigger.setAttribute('aria-expanded', open ? 'true' : 'false');
    list.hidden = !open;
    if (open) {
      document.addEventListener('mousedown', onDocMouse);
      document.addEventListener('keydown', onDocKey);
      const idx = options.findIndex((o) => o.value === opts.value);
      highlight = idx >= 0 ? idx : 0;
      scrollToHighlight();
    } else {
      document.removeEventListener('mousedown', onDocMouse);
      document.removeEventListener('keydown', onDocKey);
    }
  }

  function pick(value) {
    setOpen(false);
    if (value !== opts.value) opts.onChange(value);
    trigger.focus();
  }

  function scrollToHighlight() {
    const node = optionNodes[highlight];
    if (node instanceof HTMLElement) node.scrollIntoView({ block: 'nearest' });
  }

  function updateHighlight() {
    optionNodes.forEach((node, i) => {
      if (node instanceof HTMLElement) {
        node.classList.toggle('select__option--highlight', i === highlight);
      }
    });
  }

  function onDocMouse(e) {
    if (!root.contains(e.target)) setOpen(false);
  }

  function onDocKey(e) {
    if (e.key === 'Escape') {
      setOpen(false);
      trigger.focus();
    }
  }

  trigger.addEventListener('click', () => setOpen(!open));

  trigger.addEventListener('keydown', (e) => {
    if (opts.disabled) return;
    if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
      e.preventDefault();
      if (!open) {
        setOpen(true);
        return;
      }
      if (e.key === 'ArrowDown') highlight = Math.min(highlight + 1, options.length - 1);
      else highlight = Math.max(highlight - 1, 0);
      updateHighlight();
      scrollToHighlight();
      return;
    }
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault();
      if (!open) setOpen(true);
      else pick(options[highlight].value);
      return;
    }
    if (e.key === 'Escape' && open) {
      e.preventDefault();
      setOpen(false);
    }
  });

  root.appendChild(trigger);
  root.appendChild(list);

  return root;
}
