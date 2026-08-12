import { el } from '../lib/dom.js';

export type ChipItem = { value: string; label: string } | string;

export type ChipRowProps = {
  items: ChipItem[];
  selected: string;
  onSelect: (value: string) => void;
};

export type ChipRowHandle = {
  destroy: () => void;
  update: (newProps: { selected: string }) => void;
};

/**
 * Mount a chip row filter control inside container.
 */
export function mount(container: HTMLElement, props: ChipRowProps): ChipRowHandle {
  let activeSelected = props.selected;
  let chipElements: HTMLElement[] = [];

  const chipRow = el('div', {
    className: 'chip-row',
    role: 'radiogroup',
    'aria-label': 'Filter options',
  });

  function renderChips(): void {
    chipRow.replaceChildren();
    chipElements = [];

    props.items.forEach((item, index) => {
      const val = typeof item === 'string' ? item : item.value;
      const lbl = typeof item === 'string' ? item : item.label;
      const isSel = val === activeSelected;

      const chip = el('button', {
        type: 'button',
        className: 'chip' + (isSel ? ' chip--active' : ''),
        role: 'radio',
        'aria-checked': isSel ? 'true' : 'false',
        tabIndex: isSel ? 0 : -1,
        onClick: () => {
          activeSelected = val;
          renderChips();
          props.onSelect(val);
        },
      }, lbl);

      chip.addEventListener('keydown', (e: KeyboardEvent) => {
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
    },
  };
}
