import { useMemo } from 'react';
import type { Brand } from '../../helpers/brands_api.js';
import { useGridRowAction } from '../../helpers/use_grid_row_action.js';
import { formatLocaleDateTime } from '../../helpers/format_display.js';
import { Button } from '../system/button.js';
import { EmptyState } from '../system/empty_state.js';
import { BrandCreativesPanel } from './brand_creatives_panel.js';
import gridStyles from './brands_grid.module.css';

export type BrandsGridProps = {
  items: Brand[];
  loading: boolean;
  expandedBrandId: string | null;
  canWrite: boolean;
  onToggleExpand: (brandId: string) => void;
  onReload: () => void;
};

function shortId(value: string | undefined): string {
  if (!value) return '-';
  if (value.length <= 16) return value;
  return `${value.slice(0, 12)}...`;
}

function buildRowView(items: Brand[]) {
  const len = items.length;
  const ids = new Array<string>(len);
  const names = new Array<string>(len);
  const shortIds = new Array<string>(len);
  const updatedLabels = new Array<string>(len);
  const freqLabels = new Array<string>(len);
  for (let i = 0; i < len; i += 1) {
    const brand = items[i];
    const id = brand.id ?? '';
    ids[i] = id;
    names[i] = brand.name ?? id;
    shortIds[i] = shortId(id);
    updatedLabels[i] = formatLocaleDateTime(brand.updated_at);
    freqLabels[i] =
      brand.freq_limit != null && brand.freq_window != null
        ? `${brand.freq_limit}/${brand.freq_window}`
        : '-';
  }
  return { ids, names, shortIds, updatedLabels, freqLabels, len };
}

function SkeletonRows() {
  return (
    <>
      {Array.from({ length: 5 }, (_, index) => (
        <div key={`skel-${index}`} className={[gridStyles.dataRow, gridStyles.skeletonRow].join(' ')}>
          <span className={gridStyles.bar} />
          <span className={gridStyles.bar} />
          <span className={gridStyles.bar} />
          <span className={gridStyles.bar} />
          <span className={gridStyles.bar} />
        </div>
      ))}
    </>
  );
}

export function BrandsGrid({
  items,
  loading,
  expandedBrandId,
  canWrite,
  onToggleExpand,
  onReload,
}: BrandsGridProps) {
  const rowView = useMemo(() => buildRowView(items), [items]);
  const onExpandClick = useGridRowAction(onToggleExpand);

  return (
    <div className={gridStyles.grid} role="grid" aria-label="Brands">
      <div className={gridStyles.headerRow} role="row">
        <div className={gridStyles.headerCell} role="columnheader">
          Name
        </div>
        <div className={gridStyles.headerCell} role="columnheader">
          ID
        </div>
        <div className={gridStyles.headerCell} role="columnheader">
          Updated
        </div>
        <div className={gridStyles.headerCell} role="columnheader">
          Freq cap
        </div>
        <div className={gridStyles.headerCell} role="columnheader">
          Expand
        </div>
      </div>

      {loading && items.length === 0 ? <SkeletonRows /> : null}

      {!loading && items.length === 0 ? (
        <div className={gridStyles.emptyWrap}>
          <EmptyState message="No brands for this customer." />
        </div>
      ) : null}

      {Array.from({ length: rowView.len }, (_, index) => {
        const id = rowView.ids[index];
        const expanded = expandedBrandId === id;
        return (
          <div key={id} role="rowgroup">
            <div className={gridStyles.dataRow} role="row">
              <div className={gridStyles.nameCell} role="gridcell">
                {rowView.names[index]}
              </div>
              <div className={gridStyles.monoCell} role="gridcell">
                {rowView.shortIds[index]}
              </div>
              <div className={gridStyles.mutedCell} role="gridcell">
                {rowView.updatedLabels[index]}
              </div>
              <div className={gridStyles.mutedCell} role="gridcell">
                {rowView.freqLabels[index]}
              </div>
              <div className={gridStyles.expandCell} role="gridcell">
                <Button
                  variant="secondary"
                 
                  data-row-id={id}
                  onClick={onExpandClick}
                >
                  {expanded ? 'Hide' : 'Creatives'}
                </Button>
              </div>
            </div>
            {expanded && id ? (
              <BrandCreativesPanel brandId={id} canWrite={canWrite} onReloadBrands={onReload} />
            ) : null}
          </div>
        );
      })}
    </div>
  );
}
