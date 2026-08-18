export type CheckboxProps = {
  checked: boolean;
  disabled?: boolean;
  onChange: (checked: boolean) => void;
  label?: string;
  id?: string;
  className?: string;
};

export function Checkbox({ checked, disabled, onChange, label, id, className }: CheckboxProps) {
  const inputId = id ?? `check-${label?.replace(/\s+/g, '-').toLowerCase() ?? 'box'}`;
  return (
    <label
      className={['check', disabled ? 'check--disabled' : '', className ?? '']
        .filter(Boolean)
        .join(' ')}
    >
      <input
        type="checkbox"
        id={inputId}
        className="check__native"
        checked={checked}
        disabled={disabled}
        onChange={(e) => {
          if (!disabled) onChange(e.target.checked);
        }}
      />
      <span className="check__box" aria-hidden="true" />
      {label ? <span className="check__label">{label}</span> : null}
    </label>
  );
}
