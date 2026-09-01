import { Link } from 'react-router-dom';

export function NotFoundPage() {
  return (
    <div className="flex min-h-screen flex-col items-center justify-center gap-4 p-6 text-center">
      <h1 className="text-lg font-semibold">Page not found</h1>
      <p className="text-sm text-muted-foreground">The route does not exist in this console.</p>
      <Link className="text-sm underline" to="/">
        Go home
      </Link>
    </div>
  );
}
