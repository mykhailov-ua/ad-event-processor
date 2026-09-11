import { Input } from '@/components/ui/input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { FilterField } from '@/shell/filter_panel';

export type ExportHubNotifyChannel = 'none' | 'in_app' | 'email' | 'slack_webhook';

export type ExportHubNotifyFieldsProps = {
  disabled?: boolean;
  channel: ExportHubNotifyChannel;
  email: string;
  webhookUrl: string;
  onChannelChange: (value: ExportHubNotifyChannel) => void;
  onEmailChange: (value: string) => void;
  onWebhookUrlChange: (value: string) => void;
};

export function ExportHubNotifyFields({
  disabled,
  channel,
  email,
  webhookUrl,
  onChannelChange,
  onEmailChange,
  onWebhookUrlChange,
}: ExportHubNotifyFieldsProps) {
  return (
    <>
      <FilterField htmlFor="export-hub-notify-channel" label="Notify on completion">
        <Select
          disabled={disabled}
          value={channel}
          onValueChange={(value) => onChannelChange(value as ExportHubNotifyChannel)}
        >
          <SelectTrigger id="export-hub-notify-channel">
            <SelectValue placeholder="Select channel" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="none">None</SelectItem>
            <SelectItem value="in_app">In-app feed</SelectItem>
            <SelectItem value="email">Email</SelectItem>
            <SelectItem value="slack_webhook">Slack webhook</SelectItem>
          </SelectContent>
        </Select>
      </FilterField>
      {channel === 'email' ? (
        <FilterField htmlFor="export-hub-notify-email" label="Notify email">
          <Input
            disabled={disabled}
            id="export-hub-notify-email"
            type="email"
            value={email}
            onChange={(event) => onEmailChange(event.target.value)}
          />
        </FilterField>
      ) : null}
      {channel === 'slack_webhook' ? (
        <FilterField htmlFor="export-hub-notify-webhook" label="Slack webhook URL">
          <Input
            disabled={disabled}
            id="export-hub-notify-webhook"
            type="url"
            value={webhookUrl}
            onChange={(event) => onWebhookUrlChange(event.target.value)}
          />
        </FilterField>
      ) : null}
    </>
  );
}
