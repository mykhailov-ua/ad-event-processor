import { useEffect, useMemo, useState } from 'react';
import { Button } from './button.js';
import { Checkbox } from './checkbox.js';
import { Modal } from './modal.js';

const STRONG_TOKEN = 'DELETE';

export type ConfirmDialogProps = {
  open: boolean;
  level: string;
  title?: string;
  description?: string;
  layout?: 'vertical' | 'horizontal';
  onConfirm: () => void;
  onCancel: () => void;
};

/**
 * Registry confirm dialog.
 */
export function ConfirmDialog({
  open,
  level,
  title,
  description,
  layout,
  onConfirm,
  onCancel,
}: ConfirmDialogProps) {
  const isStrong = level === 'strong';
  const isDestructive = level === 'destructive' || level === 'financial';

  const defaultTitle = useMemo(() => {
    if (level === 'retry') return 'Retry operation?';
    if (level === 'financial') return 'Confirm financial operation';
    if (level === 'destructive') return 'Confirm action';
    if (level === 'strong') return 'Critical action';
    return 'Confirm action';
  }, [level]);

  const [typed, setTyped] = useState('');
  const [strongChecked, setStrongChecked] = useState(false);

  useEffect(() => {
    if (open) {
      setTyped('');
      setStrongChecked(false);
    }
  }, [open, level, title, description]);

  const canConfirm = isStrong ? typed === STRONG_TOKEN && strongChecked : true;
  const isVertical = layout === 'vertical' || (layout === undefined && (isDestructive || isStrong));

  const handleClose = () => {
    setTyped('');
    setStrongChecked(false);
    onCancel();
  };

  const handleConfirm = () => {
    if (!canConfirm) return;
    setTyped('');
    setStrongChecked(false);
    onConfirm();
  };

  return (
    <Modal
      open={open}
      title={title || defaultTitle}
      description={description}
      onClose={handleClose}
      footerClass={isVertical ? 'modal__footer--vertical' : undefined}
      actions={isVertical ? (
        <>
          <Button
            label="Confirm"
            variant={isDestructive || isStrong ? 'danger' : 'primary'}
            className="btn--block"
            disabled={!canConfirm}
            onClick={handleConfirm}
          />
          <Button label="Cancel" variant="ghost" className="btn--block" onClick={handleClose} />
        </>
      ) : (
        <>
          <Button label="Cancel" variant="secondary" onClick={handleClose} />
          <Button
            label="Confirm"
            variant={isDestructive || isStrong ? 'danger' : 'primary'}
            disabled={!canConfirm}
            onClick={handleConfirm}
          />
        </>
      )}
    >
      {isStrong ? (
        <div className="form-field">
          <label className="form-label" htmlFor="confirm-strong-token">
            {`Type ${STRONG_TOKEN} to confirm`}
          </label>
          <input
            id="confirm-strong-token"
            type="text"
            className="form-input"
            autoComplete="off"
            value={typed}
            onChange={(e) => setTyped(e.target.value)}
          />
          <Checkbox
            label="I understand the consequences"
            checked={strongChecked}
            onChange={setStrongChecked}
          />
        </div>
      ) : null}
    </Modal>
  );
}
