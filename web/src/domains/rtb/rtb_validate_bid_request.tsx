import { PageChrome } from '@/shell/page_chrome';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import type { OpenRtbValidationResult } from '@/api/types';
import { RtbNav, RtbLicenseStub, rtbPanelError } from '@/domains/rtb/rtb_nav';

export type RtbValidateBidRequestProps = {
  draftJson: string;
  result: OpenRtbValidationResult | undefined;
  validating: boolean;
  error: Error | undefined;
  licenseGated: boolean;
  onDraftJsonChange: (value: string) => void;
  onValidate: () => void;
};

export function RtbValidateBidRequest({
  draftJson,
  result,
  validating,
  error,
  licenseGated,
  onDraftJsonChange,
  onValidate,
}: RtbValidateBidRequestProps) {
  if (licenseGated) {
    return (
      <PageChrome title="Validate bid request">
        <RtbNav />
        <RtbLicenseStub />
      </PageChrome>
    );
  }

  return (
    <PageChrome title="Validate bid request">
      <RtbNav />

      <section className="grid max-w-2xl gap-4">
        <div className="grid gap-2">
          <Label htmlFor="rtb-validate-json">OpenRTB bid request JSON</Label>
          <Textarea
            id="rtb-validate-json"
            rows={12}
            value={draftJson}
            onChange={(event) => onDraftJsonChange(event.target.value)}
          />
        </div>
        <Button disabled={validating} onClick={onValidate} type="button">
          Validate
        </Button>
      </section>

      {error ? rtbPanelError(error, 'Validation request failed') : null}

      {result ? (
        <section className="grid gap-2 text-sm">
          <div className="flex flex-wrap gap-2">
            <Badge variant={result.valid ? 'default' : 'destructive'}>
              {result.valid ? 'valid' : 'invalid'}
            </Badge>
            {result.version ? <Badge variant="outline">v{result.version}</Badge> : null}
          </div>
          {(result.errors ?? []).length > 0 ? (
            <ul className="list-inside list-disc text-muted-foreground">
              {(result.errors ?? []).map((item) => (
                <li key={item}>{item}</li>
              ))}
            </ul>
          ) : null}
        </section>
      ) : null}
    </PageChrome>
  );
}
