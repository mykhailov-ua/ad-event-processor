import { useMemo } from 'react';

import {
  type CommandPaletteContextualAction,
  useCommandPaletteContextualRegistration,
} from '@/shell/command_palette_contextual';

type UseCampaignsCommandPaletteActionsArgs = {
  selectedCount: number;
  onPauseSelected: () => void;
  onResumeSelected: () => void;
};

export function useCampaignsCommandPaletteActions({
  selectedCount,
  onPauseSelected,
  onResumeSelected,
}: UseCampaignsCommandPaletteActionsArgs) {
  const actions = useMemo((): CommandPaletteContextualAction[] => {
    if (selectedCount <= 0) {
      return [];
    }

    const meta = `${selectedCount} selected`;

    return [
      {
        item: {
          id: 'campaigns-bulk-pause',
          kind: 'action',
          label: 'Pause selected campaigns',
          href: '/campaigns',
          meta,
          group: 'Selection',
        },
        run: onPauseSelected,
      },
      {
        item: {
          id: 'campaigns-bulk-resume',
          kind: 'action',
          label: 'Resume selected campaigns',
          href: '/campaigns',
          meta,
          group: 'Selection',
        },
        run: onResumeSelected,
      },
    ];
  }, [onPauseSelected, onResumeSelected, selectedCount]);

  useCommandPaletteContextualRegistration(actions);
}
