import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';

type StubBannerProps = {
  title?: string;
  message: string;
};

export function StubBanner({ title = 'Not available', message }: StubBannerProps) {
  return (
    <Card className="border-border/40 bg-muted/30">
      <CardHeader className="pb-2">
        <CardTitle className="text-base text-muted-foreground">{title}</CardTitle>
      </CardHeader>
      <CardContent>
        <p className="text-sm text-muted-foreground">{message}</p>
      </CardContent>
    </Card>
  );
}
