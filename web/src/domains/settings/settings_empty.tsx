import type { ReactNode } from 'react';

import {
  type SettingsEmptyField,
  SETTINGS_EMPTY_COPY,
  settingsEmptyMessage,
} from '@/lib/settings_empty_fields';
import {
  formatSettingsDisplayValue,
  usesSettingsDisplayValue,
} from '@/lib/settings_display_values';

export type { SettingsEmptyField } from '@/lib/settings_empty_fields';

export { settingsEmptyMessage };

export function settingsEmptyValue(field: SettingsEmptyField): ReactNode {
  return <span className="text-muted-foreground">{SETTINGS_EMPTY_COPY[field]}</span>;
}

export function settingsTextValue(value: string, field: SettingsEmptyField): ReactNode {
  const trimmed = value.trim();
  if (trimmed) {
    if (usesSettingsDisplayValue(field)) {
      return formatSettingsDisplayValue(trimmed, field);
    }
    return trimmed;
  }
  return settingsEmptyValue(field);
}

export function settingsMonoValue(value: string, field: SettingsEmptyField): ReactNode {
  const trimmed = value.trim();
  if (trimmed) {
    return <span className="font-mono text-xs">{trimmed}</span>;
  }
  return settingsEmptyValue(field);
}

export function settingsBoolValue(
  value: boolean | undefined,
  field: SettingsEmptyField
): ReactNode {
  if (value === undefined) {
    return settingsEmptyValue(field);
  }
  return value ? 'Enabled' : 'Disabled';
}
