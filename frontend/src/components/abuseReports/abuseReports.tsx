import { useCallback, useEffect, useState } from "react";
import { useLocale } from "../../context/LocaleContext";
import { apiUrl } from "../../lib/urls";
import { useUser } from "../../context/UserContext";
import { getUserRole } from "../../lib/userAccess";
import styles from "./abuseReports.module.css";

interface AbuseReport {
  report_id: number;
  short_code: string;
  original_url?: string;
  reporter_email?: string;
  reason:
    | "phishing"
    | "malware"
    | "spam"
    | "impersonation"
    | "illegal"
    | "other";
  details: string;
  created_at: string;
}

interface BlockedDomain {
  domain_id: number;
  domain: string;
  reason: string;
  created_at: string;
}

export default function AbuseReports() {
  const { user } = useUser();
  const { t, formatDate } = useLocale();
  const [reports, setReports] = useState<AbuseReport[]>([]);
  const [blockedDomains, setBlockedDomains] = useState<BlockedDomain[]>([]);
  const [notes, setNotes] = useState<Record<number, string>>({});
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState<number | null>(null);
  const [error, setError] = useState("");
  const role = user ? getUserRole(user) : "user";
  const canUnblock = role === "owner" || role === "admin";

  const load = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const [reportResponse, domainResponse] = await Promise.all([
        fetch(apiUrl("/api/admin/abuse-reports?status=pending&limit=50"), {
          credentials: "include",
        }),
        fetch(apiUrl("/api/admin/blocked-domains"), {
          credentials: "include",
        }),
      ]);
      if (!reportResponse.ok || !domainResponse.ok) {
        throw new Error(t("admin.abuseLoadError"));
      }
      const reportPayload = (await reportResponse.json()) as {
        data: AbuseReport[];
      };
      const domainPayload = (await domainResponse.json()) as {
        data: BlockedDomain[];
      };
      setReports(reportPayload.data);
      setBlockedDomains(domainPayload.data);
    } catch (requestError) {
      setError(
        requestError instanceof Error
          ? requestError.message
          : t("admin.abuseLoadError"),
      );
    } finally {
      setLoading(false);
    }
  }, [t]);

  useEffect(() => {
    void load();
  }, [load]);

  const resolve = async (
    report: AbuseReport,
    status: "reviewed" | "dismissed" | "blocked",
    pauseLink = false,
    blockDomain = false,
  ) => {
    setBusy(report.report_id);
    setError("");
    try {
      const response = await fetch(
        apiUrl(`/api/admin/abuse-reports/${report.report_id}`),
        {
          method: "PATCH",
          credentials: "include",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            status,
            resolution_note: notes[report.report_id] ?? "",
            pause_link: pauseLink,
            block_domain: blockDomain,
          }),
        },
      );
      if (!response.ok) throw new Error(t("admin.abuseResolveError"));
      await load();
    } catch (requestError) {
      setError(
        requestError instanceof Error
          ? requestError.message
          : t("admin.abuseResolveError"),
      );
    } finally {
      setBusy(null);
    }
  };

  const unblock = async (domain: BlockedDomain) => {
    setBusy(-domain.domain_id);
    setError("");
    try {
      const response = await fetch(
        apiUrl(`/api/admin/blocked-domains/${domain.domain_id}`),
        { method: "DELETE", credentials: "include" },
      );
      if (!response.ok) throw new Error(t("admin.abuseResolveError"));
      await load();
    } catch (requestError) {
      setError(
        requestError instanceof Error
          ? requestError.message
          : t("admin.abuseResolveError"),
      );
    } finally {
      setBusy(null);
    }
  };

  return (
    <section className={styles.section} aria-labelledby="abuse-reports-title">
      <header>
        <span>{t("admin.abuseLabel")}</span>
        <h2 id="abuse-reports-title">{t("admin.abuseTitle")}</h2>
        <p>{t("admin.abuseDescription", { count: reports.length })}</p>
      </header>

      {error && (
        <p className={styles.error} role="alert">
          {error}
        </p>
      )}

      {loading ? (
        <div className={styles.empty}>{t("admin.abuseLoading")}</div>
      ) : reports.length === 0 ? (
        <div className={styles.empty}>{t("admin.abuseEmpty")}</div>
      ) : (
        <div className={styles.reportList}>
          {reports.map((report) => (
            <article key={report.report_id} className={styles.report}>
              <div className={styles.reportMeta}>
                <span>#{report.report_id}</span>
                <strong>{t(`report.reason.${report.reason}`)}</strong>
                <time dateTime={report.created_at}>
                  {formatDate(new Date(report.created_at), {
                    dateStyle: "medium",
                    timeStyle: "short",
                  })}
                </time>
              </div>
              <div className={styles.reportBody}>
                <a
                  href={`/${report.short_code}`}
                  target="_blank"
                  rel="noreferrer"
                >
                  /{report.short_code}
                </a>
                {report.original_url && <p>{report.original_url}</p>}
                <span>{t("admin.abuseDetails")}</span>
                <blockquote>{report.details}</blockquote>
                <small>
                  {t("admin.abuseReporter")}:{" "}
                  {report.reporter_email || t("admin.abuseAnonymous")}
                </small>
              </div>
              <div className={styles.actions}>
                <label>
                  <span>{t("admin.abuseNote")}</span>
                  <input
                    value={notes[report.report_id] ?? ""}
                    onChange={(event) =>
                      setNotes((current) => ({
                        ...current,
                        [report.report_id]: event.target.value,
                      }))
                    }
                  />
                </label>
                <div>
                  <button
                    type="button"
                    disabled={busy === report.report_id}
                    onClick={() => void resolve(report, "reviewed")}
                  >
                    {t("admin.abuseReview")}
                  </button>
                  <button
                    type="button"
                    disabled={busy === report.report_id}
                    onClick={() => void resolve(report, "dismissed")}
                  >
                    {t("admin.abuseDismiss")}
                  </button>
                  <button
                    type="button"
                    disabled={busy === report.report_id}
                    onClick={() => void resolve(report, "blocked", true)}
                  >
                    {t("admin.abuseBlockLink")}
                  </button>
                  <button
                    type="button"
                    className={styles.danger}
                    disabled={busy === report.report_id}
                    onClick={() => void resolve(report, "blocked", true, true)}
                  >
                    {t("admin.abuseBlockDomain")}
                  </button>
                </div>
              </div>
            </article>
          ))}
        </div>
      )}

      <div className={styles.blocked}>
        <h3>{t("admin.blockedDomains")}</h3>
        {blockedDomains.length === 0 ? (
          <p>{t("admin.noBlockedDomains")}</p>
        ) : (
          <div>
            {blockedDomains.map((domain) => (
              <span key={domain.domain_id}>
                <b>{domain.domain}</b>
                {canUnblock && (
                  <button
                    type="button"
                    disabled={busy === -domain.domain_id}
                    onClick={() => void unblock(domain)}
                  >
                    {t("admin.unblock")}
                  </button>
                )}
              </span>
            ))}
          </div>
        )}
      </div>
    </section>
  );
}
