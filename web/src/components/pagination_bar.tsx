import { Button } from './button.js';

export type PaginationBarProps = {
  label: string;
  prevDisabled?: boolean;
  nextDisabled?: boolean;
  onPrev: () => void;
  onNext: () => void;
};

export function PaginationBar({
  label,
  prevDisabled,
  nextDisabled,
  onPrev,
  onNext,
}: PaginationBarProps) {
  return (
    <div className="pagination-bar cluster--actions">
      <Button label="Prev" variant="secondary" size="sm" disabled={prevDisabled} onClick={onPrev} />
      <span className="text-muted text-xs pagination-bar__label">{label}</span>
      <Button label="Next" variant="secondary" size="sm" disabled={nextDisabled} onClick={onNext} />
    </div>
  );
}
