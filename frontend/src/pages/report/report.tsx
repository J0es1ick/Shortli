import { useEffect, useState, type FormEvent } from "react";
import { useSearchParams } from "react-router-dom";
import Footer from "../../components/UI/footer/footer";
import { Header } from "../../components/UI/header/header";
import { useLocale } from "../../context/LocaleContext";
import { apiUrl } from "../../lib/urls";
import styles from "./report.module.css";

const reasons = [
  "phishing",
  "malware",
  "spam",
  "impersonation",
  "illegal",
  "other",
] as const;

export default function ReportPage() {
  const { t } = useLocale();
  const [searchParams] = useSearchParams();
  const [shortLink, setShortLink] = useState(searchParams.get("link") ?? "");
  const [email, setEmail] = useState("");
  const [reason, setReason] = useState<(typeof reasons)[number]>("phishing");
  const [details, setDetails] = useState("");
  const [companyWebsite, setCompanyWebsite] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");
  const [submitted, setSubmitted] = useState(false);

  useEffect(() => {
    document.title = `${t("report.pageTitle")} — Shortli`;
  }, [t]);

  const submit = async (event: FormEvent) => {
    event.preventDefault();
    setSubmitting(true);
    setError("");
    try {
      const response = await fetch(apiUrl("/api/abuse-reports"), {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          short_link: shortLink,
          reporter_email: email || undefined,
          reason,
          details,
          company_website: companyWebsite,
        }),
      });
      const payload = (await response.json().catch(() => null)) as {
        error?: string;
      } | null;
      if (!response.ok) {
        throw new Error(payload?.error || t("report.error"));
      }
      setSubmitted(true);
    } catch (requestError) {
      setError(
        requestError instanceof Error
          ? requestError.message
          : t("report.error"),
      );
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className={styles.page}>
      <Header />
      <main className={styles.main}>
        <header className={styles.hero}>
          <span>{t("report.label")}</span>
          <h1>{t("report.title")}</h1>
          <p>{t("report.intro")}</p>
        </header>

        <section className={styles.workspace}>
          <aside>
            <span>01 / {t("report.processLabel")}</span>
            <h2>{t("report.processTitle")}</h2>
            <p>{t("report.processDescription")}</p>
            <p>{t("report.privacyNote")}</p>
          </aside>

          {submitted ? (
            <div className={styles.success} role="status">
              <span>REPORT / RECEIVED</span>
              <h2>{t("report.successTitle")}</h2>
              <p>{t("report.successDescription")}</p>
              <button type="button" onClick={() => setSubmitted(false)}>
                {t("report.another")}
              </button>
            </div>
          ) : (
            <form onSubmit={submit} className={styles.form}>
              <label>
                <span>{t("report.linkLabel")}</span>
                <input
                  required
                  value={shortLink}
                  onChange={(event) => setShortLink(event.target.value)}
                  placeholder="https://short.li/example"
                />
              </label>
              <div className={styles.row}>
                <label>
                  <span>{t("report.reasonLabel")}</span>
                  <select
                    value={reason}
                    onChange={(event) =>
                      setReason(event.target.value as typeof reason)
                    }
                  >
                    {reasons.map((item) => (
                      <option key={item} value={item}>
                        {t(`report.reason.${item}`)}
                      </option>
                    ))}
                  </select>
                </label>
                <label>
                  <span>{t("report.emailLabel")}</span>
                  <input
                    type="email"
                    value={email}
                    onChange={(event) => setEmail(event.target.value)}
                    placeholder="name@example.com"
                  />
                </label>
              </div>
              <label>
                <span>{t("report.detailsLabel")}</span>
                <textarea
                  required
                  minLength={10}
                  maxLength={2000}
                  value={details}
                  onChange={(event) => setDetails(event.target.value)}
                  placeholder={t("report.detailsPlaceholder")}
                />
                <small>{details.length} / 2000</small>
              </label>
              <label className={styles.honeypot} aria-hidden="true">
                Company website
                <input
                  tabIndex={-1}
                  autoComplete="off"
                  value={companyWebsite}
                  onChange={(event) => setCompanyWebsite(event.target.value)}
                />
              </label>
              {error && (
                <p className={styles.error} role="alert">
                  {error}
                </p>
              )}
              <button type="submit" disabled={submitting}>
                {submitting ? t("report.submitting") : t("report.submit")}
              </button>
            </form>
          )}
        </section>
      </main>
      <Footer />
    </div>
  );
}
