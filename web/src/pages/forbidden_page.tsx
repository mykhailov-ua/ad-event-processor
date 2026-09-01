import { Link } from 'react-router-dom';

export function ForbiddenPage() {
  return (
    <div className="flex min-h-screen flex-col items-center justify-center gap-4 p-6 text-center">
      <h1 className="text-lg font-semibold">Forbidden</h1>
      <p className="text-sm text-muted-foreground">You do not have access to this resource.</p>
      <Link className="text-sm underline" to="/">
        Go home
      </Link>
    </div>
  );
}
