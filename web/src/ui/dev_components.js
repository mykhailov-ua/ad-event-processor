import { el } from '../lib/dom.js';
import { renderSectionCard } from './section_card.js';
import { mount as mountChipRow } from './chip_row.js';
import { mount as mountSegmented } from './segmented_control.js';
import { renderStatusHint } from './status_hint.js';

/**
 * Mounts the /dev/components preview gallery.
 * 
 * @param {HTMLElement} container
 * @returns {{ destroy: () => void }}
 */
export function mount(container) {
  const root = el('div', {
    className: 'dev-gallery',
    style: { display: 'flex', flexDirection: 'column', gap: '24px', padding: '24px' }
  });

  root.appendChild(el('h1', { style: { fontSize: '24px', fontWeight: '600', marginBottom: '8px' } }, 'UI Component Gallery'));

  const cardSection = renderSectionCard({
    title: 'Section Cards (DiiaCard)',
    desc: 'Flat layout components with border-left highlighting based on urgency levels.',
    children: [
      el('div', { style: { display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(280px, 1fr))', gap: '16px' } },
        renderSectionCard({
          title: 'Normal/Success Card',
          urgent: 'normal',
          children: el('p', null, 'This is a success/normal urgency card.')
        }),
        renderSectionCard({
          title: 'Warning Card',
          urgent: 'warning',
          children: el('p', null, 'This is a warning urgency card.')
        }),
        renderSectionCard({
          title: 'Danger Card',
          urgent: 'danger',
          children: el('p', null, 'This is a high urgency/danger card.')
        })
      )
    ]
  });
  root.appendChild(cardSection);

  const chipContainer = el('div');
  const chipSection = renderSectionCard({
    title: 'Chip Row',
    desc: 'Horizontal scrolling selection row with keyboard arrow accessibility.',
    children: chipContainer
  });
  root.appendChild(chipSection);

  let activeFilter = 'active';
  const chipRowHandle = mountChipRow(chipContainer, {
    items: [
      { value: 'all', label: 'All Campaigns' },
      { value: 'active', label: 'Active' },
      { value: 'paused', label: 'Paused' },
      { value: 'failed', label: 'Failed' },
      { value: 'draft', label: 'Draft' }
    ],
    selected: activeFilter,
    onSelect: (val) => {
      activeFilter = val;
    }
  });

  const segmentedContainer = el('div');
  const segmentedSection = renderSectionCard({
    title: 'Segmented Control',
    desc: 'Tab-like binary/multiple selector with sliding pill layout and 160ms transition.',
    children: segmentedContainer
  });
  root.appendChild(segmentedSection);

  let activePeriod = '24h';
  const segmentedHandle = mountSegmented(segmentedContainer, {
    items: [
      { value: '1h', label: 'Last Hour' },
      { value: '24h', label: '24 Hours' },
      { value: '7d', label: '7 Days' },
      { value: '30d', label: '30 Days' }
    ],
    selected: activePeriod,
    onChange: (val) => {
      activePeriod = val;
    }
  });

  const hintSection = renderSectionCard({
    title: 'Status Hint',
    desc: 'Persistent inline feedback notices for form fields and settings panels.',
    children: el('div', { style: { display: 'flex', flexDirection: 'column', gap: '12px' } },
      renderStatusHint({ tone: 'info', message: 'Platform version mismatch has been resolved.' }),
      renderStatusHint({ tone: 'success', message: 'Wallet balance successfully synchronized with billing service.' }),
      renderStatusHint({ tone: 'error', message: 'Campaign budget cannot exceed the billing account threshold.' })
    )
  });
  root.appendChild(hintSection);

  container.replaceChildren(root);

  return {
    destroy() {
      chipRowHandle.destroy();
      segmentedHandle.destroy();
      root.remove();
    }
  };
}
