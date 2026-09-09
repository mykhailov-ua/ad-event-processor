import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react';

import type { CommandPaletteItem } from '@/api/command_palette_api';

export type CommandPaletteContextualAction = {
  item: CommandPaletteItem;
  run: () => void;
};

type CommandPaletteContextualContextValue = {
  actions: CommandPaletteContextualAction[];
  setActions: (actions: CommandPaletteContextualAction[]) => void;
  resolveRun: (id: string) => (() => void) | undefined;
};

const CommandPaletteContextualContext = createContext<CommandPaletteContextualContextValue | null>(
  null
);

export function CommandPaletteContextualProvider({ children }: { children: ReactNode }) {
  const [actions, setActions] = useState<CommandPaletteContextualAction[]>([]);

  const resolveRun = useCallback(
    (id: string) => {
      const match = actions.find((action) => action.item.id === id);
      return match?.run;
    },
    [actions]
  );

  const value = useMemo(
    () => ({
      actions,
      setActions,
      resolveRun,
    }),
    [actions, resolveRun]
  );

  return (
    <CommandPaletteContextualContext.Provider value={value}>
      {children}
    </CommandPaletteContextualContext.Provider>
  );
}

export function useCommandPaletteContextualState() {
  const ctx = useContext(CommandPaletteContextualContext);
  if (!ctx) {
    throw new Error('useCommandPaletteContextualState requires CommandPaletteContextualProvider');
  }
  return ctx;
}

export function useCommandPaletteContextualRegistration(actions: CommandPaletteContextualAction[]) {
  const { setActions } = useCommandPaletteContextualState();

  const actionKey = actions.map((action) => action.item.id).join('\0');

  useEffect(() => {
    setActions(actions);
    return () => {
      setActions([]);
    };
  }, [actionKey, actions, setActions]);
}
