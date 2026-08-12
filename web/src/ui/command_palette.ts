export type CommandPaletteHandle = {
  destroy: () => void;
  open: (initialQuery?: string) => void;
};

export type CommandPaletteDeps = {
  focusSearch: (initialQuery?: string) => void;
};

/**
 * Global Cmd/Ctrl+K shortcut — opens the sidebar search dropdown.
 */
export function installCommandPalette(deps: CommandPaletteDeps): CommandPaletteHandle {
  function onGlobalKey(e: KeyboardEvent): void {
    if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
      e.preventDefault();
      deps.focusSearch('');
    }
  }

  document.addEventListener('keydown', onGlobalKey);

  return {
    open: deps.focusSearch,
    destroy: () => {
      document.removeEventListener('keydown', onGlobalKey);
    },
  };
}
