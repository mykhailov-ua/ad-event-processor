import { ListTree, Sparkles, Workflow, Zap } from 'lucide-react';
import type { LucideIcon } from 'lucide-react';

import { BentoCard, BentoGrid, bentoToneFromKey } from '@/components/system/bento_card';

const PRESET_ICONS: LucideIcon[] = [Sparkles, Workflow, Zap, ListTree];

export type PresetCatalogRow = {
  key?: string;
  title?: string;
  description?: string;
  parameters_schema?: unknown[] | null;
};

export function PresetCatalogGrid({ items }: { items: PresetCatalogRow[] }) {
  return (
    <BentoGrid>
      {items.map((row, index) => {
        const key = row.key ?? `preset-${index}`;
        const Icon = PRESET_ICONS[index % PRESET_ICONS.length];
        const paramCount = row.parameters_schema?.length ?? 0;

        return (
          <BentoCard
            key={key}
            description={row.description}
            icon={Icon}
            meta={`${paramCount} parameter${paramCount === 1 ? '' : 's'}`}
            title={row.title ?? row.key ?? 'Preset'}
            tone={bentoToneFromKey(key)}
          />
        );
      })}
    </BentoGrid>
  );
}
