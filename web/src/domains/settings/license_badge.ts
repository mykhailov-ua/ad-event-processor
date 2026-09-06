export function licenseBadgeVariant(
  state: string
): 'default' | 'secondary' | 'destructive' | 'outline' {
  const normalized = state.toLowerCase();
  if (normalized === 'active' || normalized === 'valid') {
    return 'default';
  }
  if (normalized === 'trial' || normalized === 'grace') {
    return 'secondary';
  }
  if (normalized === 'expired' || normalized === 'revoked' || normalized === 'missing') {
    return 'destructive';
  }
  return 'outline';
}
