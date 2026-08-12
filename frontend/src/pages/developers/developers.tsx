import {
  useCallback,
  useEffect,
  useMemo,
  useState,
  type FormEvent,
} from "react";
import { Link } from "react-router-dom";
import Footer from "../../components/UI/footer/footer";
import { Header } from "../../components/UI/header/header";
import { useLocale } from "../../context/LocaleContext";
import { useUser } from "../../context/UserContext";
import { API_BASE, apiUrl } from "../../lib/urls";
import styles from "./developers.module.css";

interface APIKeyRecord {
  key_id: number;
  name: string;
  prefix: string;
  last_used_at: string | null;
  created_at: string;
  revoked_at?: string | null;
}

export default function DevelopersPage() {
  const { user, isLoading } = useUser();
  const { t, formatDate } = useLocale();
  const [keys, setKeys] = useState<APIKeyRecord[]>([]);
  const [name, setName] = useState("");
  const [secret, setSecret] = useState("");
  const [loadingKeys, setLoadingKeys] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [copied, setCopied] = useState("");

  const baseURL = useMemo(
    () => `${API_BASE || window.location.origin}/api/v1`,
    [],
  );
  const curlExample = `curl -X POST '${baseURL}/links' \\\n+  -H 'Content-Type: application/json' \\\n+  -H 'X-API-Key: YOUR_API_KEY' \\\n+  -d '{"original_url":"https://example.com/long-page","expires_at":"2026-08-01T12:00:00Z"}'`;

  const fetchKeys = useCallback(async () => {
    if (!user) return;
    setLoadingKeys(true);
    try {
      const response = await fetch(apiUrl("/api/developer/keys"), {
        credentials: "include",
      });
      if (!response.ok) throw new Error(t("developers.keysLoadError"));
      const data = await response.json();
      setKeys(data.data);
    } catch (requestError) {
      setError(
        requestError instanceof Error
          ? requestError.message
          : t("developers.keysLoadError"),
      );
    } finally {
      setLoadingKeys(false);
    }
  }, [t, user]);

  useEffect(() => {
    document.title = `${t("developers.pageTitle")} — Shortli`;
  }, [t]);
  useEffect(() => {
    void fetchKeys();
  }, [fetchKeys]);

  const createKey = async (event: FormEvent) => {
    event.preventDefault();
    if (name.trim().length < 2) return;
    setSaving(true);
    setError("");
    try {
      const response = await fetch(apiUrl("/api/developer/keys"), {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({ name: name.trim() }),
      });
      const data = await response.json().catch(() => ({}));
      if (!response.ok)
        throw new Error(data.error || t("developers.keyCreateError"));
      setSecret(data.key);
      setName("");
      await fetchKeys();
    } catch (requestError) {
      setError(
        requestError instanceof Error
          ? requestError.message
          : t("developers.keyCreateError"),
      );
    } finally {
      setSaving(false);
    }
  };

  const revokeKey = async (id: number) => {
    setError("");
    const response = await fetch(apiUrl(`/api/developer/keys/${id}`), {
      method: "DELETE",
      credentials: "include",
    });
    if (!response.ok) {
      setError(t("developers.keyRevokeError"));
      return;
    }
    await fetchKeys();
  };

  const copy = async (value: string, label: string) => {
    await navigator.clipboard.writeText(value);
    setCopied(label);
    window.setTimeout(() => setCopied(""), 1600);
  };

  const endpoints = [
    ["POST", "/links", t("developers.endpoint.create")],
    ["GET", "/links/{shortCode}", t("developers.endpoint.read")],
    ["PATCH", "/links/{shortCode}", t("developers.endpoint.update")],
    ["DELETE", "/links/{shortCode}", t("developers.endpoint.delete")],
    ["GET", "/links/{shortCode}/analytics", t("developers.endpoint.analytics")],
  ];

  return (
    <div className={styles.page}>
      <Header />
      <main className={styles.main}>
        <section className={styles.hero}>
          <span>{t("developers.label")}</span>
          <h1>{t("developers.title")}</h1>
          <p>{t("developers.intro")}</p>
          <div className={styles.hero_meta}>
            <span>REST / JSON</span>
            <span>V1</span>
            <span>120 RPM / KEY</span>
          </div>
        </section>

        <section className={styles.docs}>
          <div className={styles.section_title}>
            <span>01</span>
            <div>
              <h2>{t("developers.quickStart")}</h2>
              <p>{t("developers.quickStartDescription")}</p>
            </div>
          </div>
          <div className={styles.base_url}>
            <span>{t("developers.baseUrl")}</span>
            <code>{baseURL}</code>
            <button type="button" onClick={() => void copy(baseURL, "url")}>
              {copied === "url" ? t("common.copied") : t("common.copy")}
            </button>
          </div>
          <div className={styles.code_block}>
            <div>
              <span>cURL</span>
              <button
                type="button"
                onClick={() => void copy(curlExample, "curl")}
              >
                {copied === "curl" ? t("common.copied") : t("common.copy")}
              </button>
            </div>
            <pre>
              <code>{curlExample}</code>
            </pre>
          </div>
        </section>

        <section className={styles.endpoint_section}>
          <div className={styles.section_title}>
            <span>02</span>
            <div>
              <h2>{t("developers.endpoints")}</h2>
              <p>{t("developers.endpointsDescription")}</p>
            </div>
          </div>
          <div className={styles.endpoint_list}>
            {endpoints.map(([method, path, description]) => (
              <article key={`${method}-${path}`}>
                <strong data-method={method}>{method}</strong>
                <code>{path}</code>
                <p>{description}</p>
              </article>
            ))}
          </div>
          <a
            className={styles.schema_link}
            href={apiUrl("/api/v1/openapi.json")}
            target="_blank"
            rel="noreferrer"
          >
            {t("developers.openApi")} ↗
          </a>
        </section>

        <section className={styles.keys_section} id="api-keys">
          <div className={styles.section_title}>
            <span>03</span>
            <div>
              <h2>{t("developers.keys")}</h2>
              <p>{t("developers.keysDescription")}</p>
            </div>
          </div>
          {isLoading ? (
            <p className={styles.muted}>{t("common.loading")}</p>
          ) : !user ? (
            <div className={styles.signin_prompt}>
              <p>{t("developers.signInRequired")}</p>
              <Link to="/" state={{ openLoginModal: true }}>
                {t("header.signIn")}
              </Link>
            </div>
          ) : (
            <div className={styles.key_workspace}>
              <form onSubmit={createKey}>
                <label htmlFor="key-name">{t("developers.keyName")}</label>
                <div>
                  <input
                    id="key-name"
                    value={name}
                    onChange={(event) => setName(event.target.value)}
                    minLength={2}
                    maxLength={60}
                    placeholder={t("developers.keyPlaceholder")}
                  />
                  <button
                    type="submit"
                    disabled={saving || name.trim().length < 2}
                  >
                    {saving
                      ? t("developers.creating")
                      : t("developers.createKey")}
                  </button>
                </div>
              </form>
              {secret && (
                <div className={styles.secret} role="status">
                  <span>{t("developers.secretOnce")}</span>
                  <code>{secret}</code>
                  <button
                    type="button"
                    onClick={() => void copy(secret, "secret")}
                  >
                    {copied === "secret"
                      ? t("common.copied")
                      : t("common.copy")}
                  </button>
                </div>
              )}
              {error && (
                <p className={styles.error} role="alert">
                  {error}
                </p>
              )}
              <div className={styles.key_list}>
                {loadingKeys ? (
                  <p className={styles.muted}>{t("common.loading")}</p>
                ) : keys.length === 0 ? (
                  <p className={styles.muted}>{t("developers.noKeys")}</p>
                ) : (
                  keys.map((key) => (
                    <article key={key.key_id}>
                      <div>
                        <strong>{key.name}</strong>
                        <code>{key.prefix}••••••••</code>
                      </div>
                      <dl>
                        <div>
                          <dt>{t("developers.created")}</dt>
                          <dd>{formatDate(key.created_at)}</dd>
                        </div>
                        <div>
                          <dt>{t("developers.lastUsed")}</dt>
                          <dd>
                            {key.last_used_at
                              ? formatDate(key.last_used_at)
                              : t("developers.never")}
                          </dd>
                        </div>
                      </dl>
                      <button
                        type="button"
                        onClick={() => void revokeKey(key.key_id)}
                      >
                        {t("developers.revoke")}
                      </button>
                    </article>
                  ))
                )}
              </div>
            </div>
          )}
        </section>

        <p className={styles.security_note}>{t("developers.securityNote")}</p>
      </main>
      <Footer />
    </div>
  );
}
