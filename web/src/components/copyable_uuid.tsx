import { copyToClipboard } from '../lib/copy_text.js';
import { Icon } from './icon.js';

export type CopyableUuidProps = {
  uuid: string;
  className?: string;
};

export function CopyableUuid({ uuid, className }: CopyableUuidProps) {
  if (!uuid) return <span className="text-muted">—</span>;
  const truncated = uuid.length > 16 ? `${uuid.slice(0, 8)}…${uuid.slice(-8)}` : uuid;
  return (
    <button
      type="button"
      className={`copyable-btn ${className ?? ''}`.trim()}
      title={`Click to copy: ${uuid}`}
      aria-label={`Copy ${uuid}`}
      onClick={async (e) => {
        e.stopPropagation();
        e.preventDefault();
        await copyToClipboard(uuid);
      }}
    >
      <span className="copyable-btn__text font-mono text-hint">{truncated}</span>
      <Icon name="copy" size={13} className="copyable-btn__icon" />
      <Icon name="check" size={13} className="copyable-btn__check" />
    </button>
  );
}
