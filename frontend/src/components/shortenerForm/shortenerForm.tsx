import { useState, type FormEvent } from "react";
import { apiUrl, buildShortUrl } from "../../lib/urls";
import { useLocale } from "../../context/LocaleContext";
import QRCustomizerModal from "../UI/qrCustomizerModal/qrCustomizerModal";
import ShareModal from "../UI/shareModal/shareModal";
import styles from "./shortenerForm.module.css";

interface ShortenResponse {
  original_url: string;
  short_code: string;
  short_url: string;
  qr_code_base64: string;
  expires_at: string | null;
  is_active: boolean;
}

type Lifetime = "never" | "day" | "week" | "month" | "custom";

export default function ShortenerForm() {
  const { t, apiError } = useLocale();
  const [url, setUrl] = useState("");
  const [customAlias, setCustomAlias] = useState("");
  const [lifetime, setLifetime] = useState<Lifetime>("never");
  const [customExpiry, setCustomExpiry] = useState("");
  const [result, setResult] = useState<ShortenResponse | null>(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const [copied, setCopied] = useState(false);
  const [qrModalOpen, setQrModalOpen] = useState(false);
  const [shareModalOpen, setShareModalOpen] = useState(false);
  const [customQR, setCustomQR] = useState("");

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const trimmedUrl = url.trim();
    const trimmedAlias = customAlias.trim().toLowerCase();

    if (!trimmedUrl) {
      setError(t("shortener.required"));
      return;
    }

    if (trimmedAlias && !/^[a-z0-9_-]{3,32}$/.test(trimmedAlias)) {
      setError(t("shortener.aliasInvalid"));
      return;
    }

    let expiresAt: string | undefined;
    if (lifetime === "custom") {
      const customDate = new Date(customExpiry);
      if (
        !customExpiry ||
        Number.isNaN(customDate.getTime()) ||
        customDate.getTime() < Date.now() + 5 * 60_000
      ) {
        setError(t("shortener.expirationInvalid"));
        return;
      }
      expiresAt = customDate.toISOString();
    } else if (lifetime !== "never") {
      const days = lifetime === "day" ? 1 : lifetime === "week" ? 7 : 30;
      expiresAt = new Date(Date.now() + days * 86_400_000).toISOString();
    }

    setLoading(true);
    setError("");
    setCopied(false);

    try {
      const response = await fetch(apiUrl("/api/shorten"), {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          original_url: trimmedUrl,
          custom_alias: trimmedAlias || undefined,
          expires_at: expiresAt,
        }),
        credentials: "include",
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(apiError(errorData.error, "shortener.failed"));
      }

      const data: ShortenResponse = await response.json();
      setResult(data);
      setCustomQR(data.qr_code_base64);
      setUrl("");
      setCustomAlias("");
      window.dispatchEvent(new CustomEvent("shortli:url-created"));
    } catch (requestError) {
      setError(
        requestError instanceof Error
          ? requestError.message
          : t("shortener.genericError"),
      );
      setResult(null);
    } finally {
      setLoading(false);
    }
  };

  const fullShortUrl = result
    ? buildShortUrl(result.short_code, result.short_url)
    : "";

  const handleCopy = async () => {
    if (!fullShortUrl) return;
    try {
      await navigator.clipboard.writeText(fullShortUrl);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1800);
    } catch {
      setError(t("shortener.clipboardUnavailable"));
    }
  };

  return (
    <section
      className={styles.form_shell}
      id="shorten"
      aria-label={t("shortener.aria")}
    >
      <form className={styles.form} onSubmit={handleSubmit} noValidate>
        <label htmlFor="long-url">{t("shortener.label")}</label>
        <div className={styles.input_row}>
          <input
            id="long-url"
            type="url"
            inputMode="url"
            autoComplete="url"
            value={url}
            onChange={(event) => setUrl(event.target.value)}
            placeholder={t("shortener.placeholder")}
            disabled={loading}
            aria-describedby={error ? "shortener-error" : "shortener-note"}
          />
          <button type="submit" disabled={loading}>
            <span>
              {loading ? t("shortener.working") : t("shortener.submit")}
            </span>
            <span aria-hidden="true">↗</span>
          </button>
        </div>
        <p id="shortener-note" className={styles.note}>
          {t("shortener.note")}
        </p>
        <div className={styles.alias_block}>
          <div className={styles.alias_header}>
            <label htmlFor="custom-alias">{t("shortener.aliasLabel")}</label>
            <span>{t("shortener.aliasOptional")}</span>
          </div>
          <div className={styles.alias_input}>
            <span aria-hidden="true">shortli/</span>
            <input
              id="custom-alias"
              type="text"
              inputMode="url"
              autoComplete="off"
              value={customAlias}
              onChange={(event) =>
                setCustomAlias(event.target.value.toLowerCase())
              }
              placeholder={t("shortener.aliasPlaceholder")}
              minLength={3}
              maxLength={32}
              pattern="[a-z0-9_-]{3,32}"
              disabled={loading}
              aria-describedby="custom-alias-note"
            />
          </div>
          <p id="custom-alias-note">{t("shortener.aliasNote")}</p>
        </div>
        <fieldset className={styles.lifetime_block}>
          <legend>{t("shortener.lifetimeLabel")}</legend>
          <div className={styles.lifetime_options}>
            {(["never", "day", "week", "month", "custom"] as Lifetime[]).map(
              (value) => (
                <button
                  key={value}
                  type="button"
                  className={lifetime === value ? styles.selected : ""}
                  onClick={() => setLifetime(value)}
                  aria-pressed={lifetime === value}
                  disabled={loading}
                >
                  {t(`shortener.lifetime.${value}`)}
                </button>
              ),
            )}
          </div>
          {lifetime === "custom" && (
            <label className={styles.custom_expiry}>
              <span>{t("shortener.customExpiry")}</span>
              <input
                type="datetime-local"
                value={customExpiry}
                min={new Date(Date.now() + 5 * 60_000)
                  .toISOString()
                  .slice(0, 16)}
                onChange={(event) => setCustomExpiry(event.target.value)}
                disabled={loading}
              />
            </label>
          )}
          <p>{t("shortener.lifetimeNote")}</p>
        </fieldset>
        {error && (
          <p id="shortener-error" className={styles.error} role="alert">
            {error}
          </p>
        )}
      </form>

      {result && (
        <div className={styles.result} aria-live="polite">
          <div className={styles.result_copy}>
            <span className={styles.result_status}>
              {t("shortener.linkReady")}
            </span>
            <a href={fullShortUrl} target="_blank" rel="noopener noreferrer">
              {fullShortUrl.replace(/^https?:\/\//, "")}
            </a>
            <p title={result.original_url}>{result.original_url}</p>
            <div className={styles.actions}>
              <button type="button" onClick={handleCopy}>
                {copied ? t("common.copied") : t("shortener.copyLink")}
              </button>
              <button type="button" onClick={() => setShareModalOpen(true)}>
                {t("common.share")}
              </button>
              <a
                href={customQR || result.qr_code_base64}
                download={`shortli-${result.short_code}.png`}
              >
                {t("shortener.downloadQr")}
              </a>
            </div>
          </div>
          <button
            type="button"
            className={styles.qr_button}
            onClick={() => setQrModalOpen(true)}
            aria-label={t("shortener.customizeQr")}
          >
            <img
              src={customQR || result.qr_code_base64}
              alt={t("shortener.qrAlt")}
            />
            <span>{t("shortener.customize")}</span>
          </button>
        </div>
      )}

      <QRCustomizerModal
        isOpen={qrModalOpen}
        onClose={() => setQrModalOpen(false)}
        qrCode={
          result
            ? { base64: result.qr_code_base64, shortCode: result.short_code }
            : null
        }
        onUpdateQRCode={setCustomQR}
      />
      <ShareModal
        isOpen={shareModalOpen}
        onClose={() => setShareModalOpen(false)}
        url={fullShortUrl}
        title={t("shortener.shareTitle")}
        description={t("shortener.shareDescription", { url: fullShortUrl })}
      />
    </section>
  );
}
