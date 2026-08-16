import { pushToastMessage } from '../helpers/toast_ui.js';
import { Button } from './button.js';

export type IntegrationCopyRowProps = {
  label: string;
  value: string;
  testId?: string;
};

function copyText(label: string, text: string): void {
  navigator.clipboard?.writeText(text).then(() => {
    pushToastMessage({ title: 'Copied', message: `${label} copied to clipboard` });
  }).catch(() => {
    pushToastMessage({ title: 'Copy failed', message: text || '(empty)' });
  });
}

/**
 * Copyable code block with label and copy button.
 */
export function IntegrationCopyRow({ label, value, testId }: IntegrationCopyRowProps) {
  return (
    <div className="integration-copy-row" data-testid={testId}>
      <div className="integration-copy-row__head">
        <span className="form-label">{label}</span>
        <Button
          label="Copy"
          variant="secondary"
          size="sm"
          data-testid={testId ? `${testId}-copy` : undefined}
          onClick={() => copyText(label, value)}
        />
      </div>
      <code className="code-block">{value}</code>
    </div>
  );
}
