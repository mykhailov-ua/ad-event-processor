import { el } from '../lib/dom.js';

/**
 * Mounts a ChipRow component inside container.
 * 
 * @param {HTMLElement} container
 * @param {{ items: Array<{ value: string, label: string }|string>, selected: string, onSelect: (value: string) => void }} props
 * @returns {{ destroy: () => void, update: (newProps: { selected: string }) => void }}
 */
export function mount(container, props) {
  let activeSelected = props.selected;
  let chipElements = [];

  const chipRow = el('div', {
    className: 'chip-row',
    role: 'radiogroup',
    'aria-label': 'Filter options'
  });

  function renderChips() {
    chipRow.replaceChildren();
    chipElements = [];

    props.items.forEach((item, index) => {
      const val = typeof item === 'string' ? item : item.value;
      const lbl = typeof item === 'string' ? item : item.label;
      const isSel = val === activeSelected;

      const chip = el('button', {
        type: 'button',
        className: 'chip' + (isSel ? ' chip--selected' : ''),
        role: 'radio',
        'aria-checked': isSel ? 'true' : 'false',
        tabIndex: isSel ? 0 : -1,
        onClick: () => {
          activeSelected = val;
          renderChips();
          props.onSelect(val);
        }
      }, lbl);

      chip.addEventListener('keydown', (e) => {
        if (e.key === 'ArrowRight' || e.key === 'ArrowDown') {
          e.preventDefault();
          const nextIndex = (index + 1) % chipElements.length;
          chipElements[nextIndex].focus();
          chipElements[nextIndex].click();
        } else if (e.key === 'ArrowLeft' || e.key === 'ArrowUp') {
          e.preventDefault();
          const prevIndex = (index - 1 + chipElements.length) % chipElements.length;
          chipElements[prevIndex].focus();
          chipElements[prevIndex].click();
        }
      });

      chipElements.push(chip);
      chipRow.appendChild(chip);
    });
  }

  renderChips();
  container.replaceChildren(chipRow);

  return {
    destroy() {
      chipRow.remove();
    },
    update(newProps) {
      if (newProps.selected !== activeSelected) {
        activeSelected = newProps.selected;
        renderChips();
      }
    }
  };
}
