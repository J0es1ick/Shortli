import { useEffect, useMemo, useState } from "react";
import { createPortal } from "react-dom";
import { useLocale } from "../../../context/LocaleContext";
import { apiUrl, buildShortUrl } from "../../../lib/urls";
import styles from "./linkDetailsModal.module.css";

export interface ManagedLink {
  url_id: number;
  original_url: string;
  short_code: string;
  short_url?: string;
  click_count: number;
  created_at: string;
  expires_at: string | null;
  is_active: boolean;
}

interface Bucket {
  label: string;
  count: number;
}
interface Analytics {
  short_code: string;
  period_days: number;
  lifetime_clicks: number;
  total_clicks: number;
  unique_clicks: number;
  daily: Bucket[];
  devices: Bucket[];
  browsers: Bucket[];
  operating_systems: Bucket[];
  referrers: Bucket[];
  countries: Bucket[];
}

interface Props {
  item: ManagedLink | null;
  onClose: () => void;
  onUpdated: () => Promise<void> | void;
}

type ExpiryMode = "keep" | "never" | "day" | "week" | "month" | "custom";

export default function LinkDetailsModal({ item, onClose, onUpdated }: Props) {
  const { t, apiError, formatDate, formatNumber } = useLocale();
  const [period, setPeriod] = useState(30);
  const [analytics, setAnalytics] = useState<Analytics | null>(null);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [expiryMode, setExpiryMode] = useState<ExpiryMode>("keep");
  const [customExpiry, setCustomExpiry] = useState("");

  useEffect(() => {
    if (!item) return;
    const controller = new AbortController();
    setLoading(true);
    setError("");
    fetch(apiUrl(`/api/urls/${item.short_code}/analytics?days=${period}`), {
      credentials: "include",
      signal: controller.signal,
    })
      .then(async (response) => {
        if (!response.ok) {
          const data = await response.json().catch(() => ({}));
          throw new Error(apiError(data.error, "manage.analyticsError"));
        }
        return response.json() as Promise<Analytics>;
      })
      .then(setAnalytics)
      .catch((requestError) => {
        if (
          requestError instanceof DOMException &&
          requestError.name === "AbortError"
        )
          return;
        setError(
          requestError instanceof Error
            ? requestError.message
            : t("manage.analyticsError"),
        );
      })
      .finally(() => setLoading(false));
    return () => controller.abort();
  }, [apiError, item, period, t]);

  useEffect(() => {
    if (!item) return;
    setExpiryMode("keep");
    setCustomExpiry(
      item.expires_at
        ? new Date(item.expires_at).toISOString().slice(0, 16)
        : "",
    );
  }, [item]);

  useEffect(() => {
    if (!item) return;
    const closeOnEscape = (event: KeyboardEvent) => {
      if (event.key === "Escape") onClose();
    };
    window.addEventListener("keydown", closeOnEscape);
    return () => window.removeEventListener("keydown", closeOnEscape);
  }, [item, onClose]);

  const maxDaily = useMemo(
    () => Math.max(1, ...(analytics?.daily.map((entry) => entry.count) ?? [1])),
    [analytics],
  );

  if (!item) return null;

  const updateSettings = async (payload: Record<string, unknown>) => {
    setSaving(true);
    setError("");
    try {
      const response = await fetch(apiUrl(`/api/urls/${item.short_code}`), {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify(payload),
      });
      if (!response.ok) {
        const data = await response.json().catch(() => ({}));
        throw new Error(apiError(data.error, "manage.saveError"));
      }
      await onUpdated();
    } catch (requestError) {
      setError(
        requestError instanceof Error
          ? requestError.message
          : t("manage.saveError"),
      );
    } finally {
      setSaving(false);
    }
  };

  const saveExpiration = async () => {
    if (expiryMode === "keep") return;
    if (expiryMode === "never") {
      await updateSettings({ clear_expiration: true });
      return;
    }
    let expiresAt: Date;
    if (expiryMode === "custom") {
      expiresAt = new Date(customExpiry);
      if (
        !customExpiry ||
        Number.isNaN(expiresAt.getTime()) ||
        expiresAt.getTime() < Date.now() + 5 * 60_000
      ) {
        setError(t("shortener.expirationInvalid"));
        return;
      }
    } else {
      const days = expiryMode === "day" ? 1 : expiryMode === "week" ? 7 : 30;
      expiresAt = new Date(Date.now() + days * 86_400_000);
    }
    await updateSettings({
      expires_at: expiresAt.toISOString(),
      is_active: true,
    });
  };

  const breakdowns: Array<[string, Bucket[]]> = analytics
    ? [
        [t("manage.devices"), analytics.devices],
        [t("manage.browsers"), analytics.browsers],
        [t("manage.systems"), analytics.operating_systems],
        [t("manage.sources"), analytics.referrers],
        [t("manage.countries"), analytics.countries],
      ]
    : [];

  return createPortal(
    <div
      className={styles.backdrop}
      role="presentation"
      onMouseDown={(event) => {
        if (event.target === event.currentTarget) onClose();
      }}
    >
      <section
        className={styles.dialog}
        role="dialog"
        aria-modal="true"
        aria-labelledby="link-details-title"
      >
        <header>
          <div>
            <span>{t("manage.label")}</span>
            <h2 id="link-details-title">
              {buildShortUrl(item.short_code, item.short_url).replace(
                /^https?:\/\//,
                "",
              )}
            </h2>
            <p title={item.original_url}>{item.original_url}</p>
          </div>
          <button
            type="button"
            className={styles.close}
            onClick={onClose}
            aria-label={t("common.close")}
          >
            ×
          </button>
        </header>

        <div className={styles.content}>
          <aside className={styles.controls}>
            <div className={styles.control_heading}>
              <span>01</span>
              <h3>{t("manage.lifecycle")}</h3>
            </div>
            <div className={styles.status_row}>
              <div>
                <span>{t("manage.status")}</span>
                <strong
                  className={item.is_active ? styles.active : styles.paused}
                >
                  {item.is_active ? t("manage.active") : t("manage.paused")}
                </strong>
              </div>
              <button
                type="button"
                disabled={saving}
                onClick={() =>
                  void updateSettings({ is_active: !item.is_active })
                }
              >
                {item.is_active ? t("manage.pause") : t("manage.resume")}
              </button>
            </div>
            <p className={styles.safety_note}>
              {t("manage.destinationLocked")}
            </p>

            <label className={styles.expiry_field}>
              <span>{t("manage.expiration")}</span>
              <select
                value={expiryMode}
                onChange={(event) =>
                  setExpiryMode(event.target.value as ExpiryMode)
                }
              >
                <option value="keep">{t("manage.keepExpiration")}</option>
                <option value="never">{t("shortener.lifetime.never")}</option>
                <option value="day">{t("shortener.lifetime.day")}</option>
                <option value="week">{t("shortener.lifetime.week")}</option>
                <option value="month">{t("shortener.lifetime.month")}</option>
                <option value="custom">{t("shortener.lifetime.custom")}</option>
              </select>
            </label>
            {expiryMode === "custom" && (
              <input
                className={styles.date_input}
                type="datetime-local"
                min={new Date(Date.now() + 5 * 60_000)
                  .toISOString()
                  .slice(0, 16)}
                value={customExpiry}
                onChange={(event) => setCustomExpiry(event.target.value)}
              />
            )}
            <button
              className={styles.save}
              type="button"
              disabled={saving || expiryMode === "keep"}
              onClick={() => void saveExpiration()}
            >
              {saving ? t("manage.saving") : t("manage.saveExpiration")}
            </button>
            <p className={styles.current_expiry}>
              {item.expires_at
                ? t("manage.currentExpiration", {
                    date: formatDate(item.expires_at, {
                      dateStyle: "medium",
                      timeStyle: "short",
                    }),
                  })
                : t("manage.noExpiration")}
            </p>
          </aside>

          <div className={styles.analytics}>
            <div className={styles.analytics_heading}>
              <div className={styles.control_heading}>
                <span>02</span>
                <h3>{t("manage.analytics")}</h3>
              </div>
              <div className={styles.periods} aria-label={t("manage.period")}>
                {[7, 30, 90, 365].map((days) => (
                  <button
                    key={days}
                    type="button"
                    className={period === days ? styles.selected : ""}
                    onClick={() => setPeriod(days)}
                  >
                    {days === 365
                      ? t("manage.year")
                      : `${days}${t("manage.daySuffix")}`}
                  </button>
                ))}
              </div>
            </div>

            {error && (
              <p className={styles.error} role="alert">
                {error}
              </p>
            )}
            {loading ? (
              <div className={styles.loading}>{t("common.loading")}</div>
            ) : (
              analytics && (
                <>
                  <div className={styles.metrics}>
                    <div>
                      <span>{t("manage.periodClicks")}</span>
                      <strong>{formatNumber(analytics.total_clicks)}</strong>
                    </div>
                    <div>
                      <span>{t("manage.unique")}</span>
                      <strong>{formatNumber(analytics.unique_clicks)}</strong>
                    </div>
                    <div>
                      <span>{t("manage.lifetime")}</span>
                      <strong>{formatNumber(analytics.lifetime_clicks)}</strong>
                    </div>
                  </div>

                  <div
                    className={styles.chart}
                    aria-label={t("manage.dailyChart")}
                  >
                    {analytics.daily.length === 0 ? (
                      <p>{t("manage.noData")}</p>
                    ) : (
                      analytics.daily.map((entry) => (
                        <div
                          key={entry.label}
                          className={styles.bar_column}
                          title={`${entry.label}: ${entry.count}`}
                        >
                          <span>{entry.count}</span>
                          <i
                            style={{
                              height: `${Math.max(6, (entry.count / maxDaily) * 100)}%`,
                            }}
                          />
                          <time dateTime={entry.label}>
                            {entry.label.slice(5)}
                          </time>
                        </div>
                      ))
                    )}
                  </div>

                  <div className={styles.breakdowns}>
                    {breakdowns.map(([title, entries]) => (
                      <section key={title}>
                        <h4>{title}</h4>
                        {entries.length === 0 ? (
                          <p>{t("manage.noData")}</p>
                        ) : (
                          entries.map((entry) => (
                            <div
                              className={styles.breakdown_row}
                              key={entry.label}
                            >
                              <span>{entry.label}</span>
                              <strong>{formatNumber(entry.count)}</strong>
                            </div>
                          ))
                        )}
                      </section>
                    ))}
                  </div>
                  <p className={styles.privacy}>{t("manage.privacy")}</p>
                </>
              )
            )}
          </div>
        </div>
      </section>
    </div>,
    document.body,
  );
}
