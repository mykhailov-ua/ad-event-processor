import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';

type ErrorBlockProps = {
  title?: string;
  message: string;
};

export function ErrorBlock({ title = 'Error', message }: ErrorBlockProps) {
  return (
    <Card className="border-destructive/50 bg-destructive/5">
      <CardHeader className="pb-2">
        <CardTitle className="text-base text-destructive">{title}</CardTitle>
      </CardHeader>
      <CardContent>
        <p className="text-sm text-muted-foreground">{message}</p>
      </CardContent>
    </Card>
  );
}
