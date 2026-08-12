import { useCallback, useEffect, useState } from "react";
import Footer from "../../components/UI/footer/footer";
import { Header } from "../../components/UI/header/header";
import { useLocale } from "../../context/LocaleContext";
import { apiUrl } from "../../lib/urls";
import styles from "./infoPages.module.css";

type State = "loading" | "operational" | "degraded" | "offline";
interface HealthResponse {
  status: "operational" | "degraded";
  version: string;
  checked_at: string;
  services: { api: string; database: string };
}

export default function StatusPage() {
  const { t, formatDate } = useLocale();
  const [state, setState] = useState<State>("loading");
  const [health, setHealth] = useState<HealthResponse | null>(null);
  const [lastChecked, setLastChecked] = useState<Date | null>(null);

  const checkHealth = useCallback(async () => {
    setState((current) => (current === "loading" ? "loading" : current));
    try {
      const response = await fetch(apiUrl("/api/health"), {
        cache: "no-store",
      });
      const data = (await response.json()) as HealthResponse;
      setHealth(data);
      setState(
        response.ok && data.status === "operational"
          ? "operational"
          : "degraded",
      );
    } catch {
      setHealth(null);
      setState("offline");
    } finally {
      setLastChecked(new Date());
    }
  }, []);

  useEffect(() => {
    document.title = `${t("status.pageTitle")} — Shortli`;
  }, [t]);

  useEffect(() => {
    void checkHealth();
    const interval = window.setInterval(() => void checkHealth(), 30_000);
    return () => window.clearInterval(interval);
  }, [checkHealth]);

  const label =
    state === "loading" ? t("status.checking") : t(`status.${state}`);
  const services = [
    {
      name: t("status.interface"),
      description: t("status.interfaceDescription"),
      value: "operational",
    },
    {
      name: t("status.api"),
      description: t("status.apiDescription"),
      value:
        health?.services.api ?? (state === "offline" ? "offline" : "checking"),
    },
    {
      name: t("status.database"),
      description: t("status.databaseDescription"),
      value:
        health?.services.database ??
        (state === "offline" ? "unknown" : "checking"),
    },
  ];

  return (
    <div className={styles.page}>
      <Header />
      <main className={`${styles.main} ${styles.status_main}`}>
        <header className={styles.status_hero}>
          <div>
            <span>{t("status.label")}</span>
            <h1>{t("status.title")}</h1>
          </div>
          <div className={`${styles.overall} ${styles[state]}`}>
            <i aria-hidden="true" />
            <span>{label}</span>
          </div>
        </header>

        <section
          className={styles.service_grid}
          aria-label={t("status.services")}
        >
          {services.map((service, index) => (
            <article key={service.name}>
              <span>{String(index + 1).padStart(2, "0")}</span>
              <div>
                <h2>{service.name}</h2>
                <p>{service.description}</p>
              </div>
              <strong className={styles[service.value] ?? styles.unknown}>
                <i aria-hidden="true" />{" "}
                {t(
                  `status.value.${service.value}` as "status.value.operational",
                )}
              </strong>
            </article>
          ))}
        </section>

        <section className={styles.status_meta}>
          <div>
            <span>{t("status.liveCheck")}</span>
            <h2>{t("status.refreshTitle")}</h2>
            <p>{t("status.refreshDescription")}</p>
          </div>
          <div className={styles.check_panel}>
            <dl>
              <div>
                <dt>{t("status.lastChecked")}</dt>
                <dd>
                  {lastChecked
                    ? formatDate(lastChecked, {
                        dateStyle: "medium",
                        timeStyle: "medium",
                      })
                    : "—"}
                </dd>
              </div>
              <div>
                <dt>{t("status.apiVersion")}</dt>
                <dd>{health?.version ?? "—"}</dd>
              </div>
            </dl>
            <button type="button" onClick={() => void checkHealth()}>
              {t("status.checkNow")}
            </button>
          </div>
        </section>
        <p className={styles.status_note}>{t("status.note")}</p>
      </main>
      <Footer />
    </div>
  );
}
