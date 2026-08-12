import { setConfirmHandler } from '../helpers/confirm_ui.js';
import { mountConfirmDialog } from './confirm_dialog.js';

/**
 * Wire the global confirm handler to mount confirm dialogs.
 */
export function installConfirmHost(_root: HTMLElement): void {
  setConfirmHandler((req) => {
    return new Promise((resolve) => {
      const dialog = mountConfirmDialog({
        level: req.entry?.level ?? 'standard',
        title: req.title ?? req.entry?.label,
        description: req.description,
        onConfirm: () => {
          dialog.destroy();
          resolve(true);
        },
        onCancel: () => {
          dialog.destroy();
          resolve(false);
        },
      });
    });
  });
}
