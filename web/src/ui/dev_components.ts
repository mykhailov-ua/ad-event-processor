import { el } from '../lib/dom.js';
import { renderSectionCard } from './section_card.js';
import { mount as mountChipRow } from './chip_row.js';
import { mountFilterToolbar } from './filter_toolbar.js';
import { mount as mountSegmented } from './segmented_control.js';
import { renderStatusHint } from './status_hint.js';

/**
 * Mount the UI component gallery for development preview.
 */
export function mount(container: HTMLElement): { destroy: () => void } {
  const root = el('div', { className: 'dev-gallery' });

  root.appendChild(el('h1', null, 'UI component gallery'));

  root.appendChild(renderSectionCard({
    title: 'Section cards',
    desc: 'Raised sections with optional urgency border.',
    children: el('div', null,
      renderSectionCard({
        title: 'Normal',
        urgent: 'normal',
        children: el('p', null, 'Normal urgency.'),
      }),
      renderSectionCard({
        title: 'Warning',
        urgent: 'warning',
        children: el('p', null, 'Warning urgency.'),
      }),
      renderSectionCard({
        title: 'Danger',
        urgent: 'danger',
        children: el('p', null, 'Danger urgency.'),
      }),
    ),
  }));

  const chipContainer = el('div');
  const chipRowHandle = mountChipRow(chipContainer, {
    items: [
      { value: 'all', label: 'All' },
      { value: 'active', label: 'Active' },
      { value: 'paused', label: 'Paused' },
    ],
    selected: 'active',
    onSelect: () => {},
  });
  root.appendChild(renderSectionCard({
    title: 'Chip row',
    desc: 'Filter chips.',
    children: chipContainer,
  }));

  const filterToolbarHost = el('div');
  mountFilterToolbar(filterToolbarHost, {
    search: true,
    searchPlaceholder: 'Search…',
    chips: [
      { value: 'all', label: 'All' },
      { value: 'active', label: 'Active' },
    ],
    chipSelected: 'all',
    onChipSelect: () => {},
  });
  root.appendChild(renderSectionCard({
    title: 'Filter toolbar',
    desc: 'Search + chips.',
    children: filterToolbarHost,
  }));

  const segmentedContainer = el('div');
  const segmentedHandle = mountSegmented(segmentedContainer, {
    items: [
      { value: '1h', label: '1h' },
      { value: '24h', label: '24h' },
      { value: '7d', label: '7d' },
    ],
    selected: '24h',
    onChange: () => {},
  });
  root.appendChild(renderSectionCard({
    title: 'Segmented control',
    desc: 'Binary / multi toggle.',
    children: segmentedContainer,
  }));

  root.appendChild(renderSectionCard({
    title: 'Status hint',
    desc: 'Inline persistent messages.',
    children: el('div', null,
      renderStatusHint({ tone: 'info', message: 'Info message.' }),
      renderStatusHint({ tone: 'success', message: 'Success message.' }),
      renderStatusHint({ tone: 'error', message: 'Error message.' }),
    ),
  }));

  container.replaceChildren(root);

  return {
    destroy() {
      chipRowHandle.destroy();
      segmentedHandle.destroy();
      root.remove();
    },
  };
}
