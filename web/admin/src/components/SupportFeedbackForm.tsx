import { useEffect, useState, type FormEvent } from "react";
import {
  fetchSupportFeedbackMeta,
  submitFeedback,
  type SupportFeedbackMeta,
} from "../api";

export function SupportFeedbackForm() {
  const [meta, setMeta] = useState<SupportFeedbackMeta | null>(null);
  const [type, setType] = useState("bug");
  const [email, setEmail] = useState("");
  const [message, setMessage] = useState("");
  const [attachBundle, setAttachBundle] = useState(false);
  const [status, setStatus] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    let cancelled = false;
    fetchSupportFeedbackMeta()
      .then((m) => {
        if (!cancelled) {
          setMeta(m);
        }
      })
      .catch((err: unknown) => {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : "failed to load meta");
        }
      });
    return () => {
      cancelled = true;
    };
  }, []);

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setError("");
    setStatus("");
    setBusy(true);
    try {
      const id = await submitFeedback({
        type,
        contact_email: email,
        message,
        attach_bundle: attachBundle,
      });
      setStatus(`Feedback submitted (${id})`);
      setMessage("");
    } catch (err) {
      setError(err instanceof Error ? err.message : "submit failed");
    } finally {
      setBusy(false);
    }
  }

  return (
    <section className="panel">
      <h2>Support feedback</h2>
      {meta ? (
        <dl className="meta-grid">
          <div>
            <dt>Deployment</dt>
            <dd>{meta.deployment_id || "—"}</dd>
          </div>
          <div>
            <dt>Version</dt>
            <dd>{meta.binary_version}</dd>
          </div>
          <div>
            <dt>SKU</dt>
            <dd>{meta.sku || "—"}</dd>
          </div>
        </dl>
      ) : null}
      <form onSubmit={onSubmit}>
        <label>
          Type
          <select value={type} onChange={(e) => setType(e.target.value)}>
            <option value="bug">Bug</option>
            <option value="feature">Feature</option>
            <option value="support">Support</option>
          </select>
        </label>
        <label>
          Contact email
          <input
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            required
          />
        </label>
        <label>
          Message
          <textarea
            value={message}
            onChange={(e) => setMessage(e.target.value)}
            rows={5}
            required
          />
        </label>
        <label className="checkbox">
          <input
            type="checkbox"
            checked={attachBundle}
            onChange={(e) => setAttachBundle(e.target.checked)}
          />
          Attach diagnostic bundle
        </label>
        {error ? <p className="error">{error}</p> : null}
        {status ? <p className="success">{status}</p> : null}
        <button type="submit" disabled={busy}>
          {busy ? "Sending…" : "Send feedback"}
        </button>
      </form>
    </section>
  );
}
