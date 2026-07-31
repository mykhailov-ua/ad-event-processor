export type AppMetaLicense = {
  state: string;
  valid_until?: string;
  banner_severity?: string;
  renew_days?: number;
};

export type AppMeta = {
  binary_version: string;
  deployment_id: string;
  sku: string;
  license?: AppMetaLicense;
};

export type SupportFeedbackMeta = {
  deployment_id: string;
  binary_version: string;
  sku: string;
};

export type User = {
  id: string;
  email?: string;
  role: string;
  customer_id: string;
};

type ApiError = {
  error?: { code?: string; message?: string };
};

function csrfFromCookie(): string {
  const match = document.cookie.match(/(?:^|;\s*)csrfToken=([^;]+)/);
  return match ? decodeURIComponent(match[1]) : "";
}

async function parseJSON<T>(res: Response): Promise<T> {
  const body = (await res.json()) as T & ApiError;
  if (!res.ok) {
    const msg = body.error?.message ?? res.statusText;
    throw new Error(msg);
  }
  return body;
}

export async function fetchAppMeta(): Promise<AppMeta> {
  const res = await fetch("/api/v1/meta", { credentials: "same-origin" });
  return parseJSON<AppMeta>(res);
}

export async function fetchSupportFeedbackMeta(): Promise<SupportFeedbackMeta> {
  const res = await fetch("/api/v1/support/feedback/meta", {
    credentials: "same-origin",
  });
  return parseJSON<SupportFeedbackMeta>(res);
}

export async function fetchCurrentUser(): Promise<User | null> {
  const res = await fetch("/api/v1/auth/me", { credentials: "same-origin" });
  if (res.status === 401) {
    return null;
  }
  const body = await parseJSON<{ user: User }>(res);
  return body.user;
}

export async function login(email: string, password: string): Promise<User> {
  const res = await fetch("/api/v1/auth/login", {
    method: "POST",
    credentials: "same-origin",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, password }),
  });
  const body = await parseJSON<{ user: User }>(res);
  const csrf = res.headers.get("X-CSRF-Token");
  if (csrf) {
    document.cookie = `csrfToken=${encodeURIComponent(csrf)}; path=/`;
  }
  return body.user;
}

export async function submitFeedback(input: {
  type: string;
  contact_email: string;
  message: string;
  attach_bundle: boolean;
}): Promise<string> {
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
  };
  const csrf = csrfFromCookie();
  if (csrf) {
    headers["X-CSRF-Token"] = csrf;
  }
  const res = await fetch("/api/v1/support/feedback", {
    method: "POST",
    credentials: "same-origin",
    headers,
    body: JSON.stringify(input),
  });
  const body = await parseJSON<{ id: string }>(res);
  return body.id;
}
