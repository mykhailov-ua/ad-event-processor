import { useEffect, useState } from "react";
import { NavLink, Navigate, Route, Routes } from "react-router-dom";
import { fetchAppMeta, fetchCurrentUser, type AppMeta, type User } from "./api";
import { LicenseBanner } from "./components/LicenseBanner";
import { LoginShell } from "./components/LoginShell";
import { OpsDashboardPlaceholder } from "./components/OpsDashboardPlaceholder";
import { SupportFeedbackForm } from "./components/SupportFeedbackForm";
import "./App.css";

export default function App() {
  const [meta, setMeta] = useState<AppMeta | null>(null);
  const [user, setUser] = useState<User | null | undefined>(undefined);
  const [metaError, setMetaError] = useState("");

  useEffect(() => {
    let cancelled = false;
    fetchAppMeta()
      .then((m) => {
        if (!cancelled) {
          setMeta(m);
        }
      })
      .catch((err: unknown) => {
        if (!cancelled) {
          setMetaError(err instanceof Error ? err.message : "meta unavailable");
        }
      });
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    let cancelled = false;
    fetchCurrentUser()
      .then((u) => {
        if (!cancelled) {
          setUser(u);
        }
      })
      .catch(() => {
        if (!cancelled) {
          setUser(null);
        }
      });
    return () => {
      cancelled = true;
    };
  }, []);

  if (user === undefined) {
    return <div className="app loading">Loading…</div>;
  }

  if (!user) {
    return (
      <div className="app">
        <LicenseBanner license={meta?.license} />
        {metaError ? <p className="error banner-error">{metaError}</p> : null}
        <LoginShell
          onSuccess={() => {
            fetchCurrentUser()
              .then(setUser)
              .catch(() => setUser(null));
          }}
        />
      </div>
    );
  }

  return (
    <div className="app">
      <LicenseBanner license={meta?.license} />
      <header className="topbar">
        <strong>eSPX Admin</strong>
        <nav>
          <NavLink to="/" end>
            Home
          </NavLink>
          <NavLink to="/feedback">Feedback</NavLink>
          <NavLink to="/ops">Ops</NavLink>
        </nav>
        <span className="muted">
          {meta?.sku ? `SKU ${meta.sku}` : ""} v{meta?.binary_version ?? "?"}
        </span>
      </header>
      <main>
        <Routes>
          <Route
            path="/"
            element={
              <section className="panel">
                <h2>Welcome</h2>
                <p className="muted">
                  Bundled operator UI. Deployment{" "}
                  {meta?.deployment_id || "not configured"}.
                </p>
              </section>
            }
          />
          <Route path="/feedback" element={<SupportFeedbackForm />} />
          <Route path="/ops" element={<OpsDashboardPlaceholder />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </main>
    </div>
  );
}
