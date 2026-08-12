import { el } from '../lib/dom.js';

export type SegmentItem = { value: string; label: string } | string;

export type SegmentedControlProps = {
  items: SegmentItem[];
  selected: string;
  onChange: (value: string) => void;
};

export type SegmentedControlHandle = {
  destroy: () => void;
  update: (newProps: { selected: string }) => void;
};

/**
 * Mount a segmented control with animated pill selection.
 */
export function mount(container: HTMLElement, props: SegmentedControlProps): SegmentedControlHandle {
  let activeSelected = props.selected;
  let btnElements: HTMLElement[] = [];

  const control = el('div', {
    className: 'segmented-control',
    role: 'tablist',
  });

  const pill = el('div', { className: 'segmented-control__pill' });
  control.appendChild(pill);

  function syncPillPosition(): void {
    const activeIndex = props.items.findIndex((item) => {
      const val = typeof item === 'string' ? item : item.value;
      return val === activeSelected;
    });

    if (activeIndex !== -1 && btnElements[activeIndex]) {
      const btn = btnElements[activeIndex];
      pill.style.transform = `translateX(${btn.offsetLeft}px)`;
      pill.style.width = `${btn.offsetWidth}px`;
    }
  }

  function renderButtons(): void {
    Array.from(control.children).forEach((child) => {
      if (child !== pill) child.remove();
    });
    btnElements = [];

    props.items.forEach((item, index) => {
      const val = typeof item === 'string' ? item : item.value;
      const lbl = typeof item === 'string' ? item : item.label;
      const isSel = val === activeSelected;

      const btn = el('button', {
        type: 'button',
        className: 'segmented-control__btn' + (isSel ? ' segmented-control__btn--active' : ''),
        role: 'tab',
        'aria-selected': isSel ? 'true' : 'false',
        tabIndex: isSel ? 0 : -1,
        onClick: () => {
          if (val === activeSelected) return;
          activeSelected = val;
          renderButtons();
          props.onChange(val);
        },
      }, lbl);

      btn.addEventListener('keydown', (e: KeyboardEvent) => {
        if (e.key === 'ArrowRight' || e.key === 'ArrowDown') {
          e.preventDefault();
          const nextIndex = (index + 1) % btnElements.length;
          btnElements[nextIndex].focus();
          btnElements[nextIndex].click();
        } else if (e.key === 'ArrowLeft' || e.key === 'ArrowUp') {
          e.preventDefault();
          const prevIndex = (index - 1 + btnElements.length) % btnElements.length;
          btnElements[prevIndex].focus();
          btnElements[prevIndex].click();
        }
      });

      btnElements.push(btn);
      control.appendChild(btn);
    });

    requestAnimationFrame(syncPillPosition);
  }

  renderButtons();
  container.replaceChildren(control);

  const resizeObserver = new ResizeObserver(() => {
    syncPillPosition();
  });
  resizeObserver.observe(control);

  return {
    destroy() {
      resizeObserver.disconnect();
      control.remove();
    },
    update(newProps) {
      if (newProps.selected !== activeSelected) {
        activeSelected = newProps.selected;
        renderButtons();
      }
    },
  };
}
