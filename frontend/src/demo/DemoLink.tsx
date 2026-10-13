import { useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { Header } from "../components/UI/header/header";
import Footer from "../components/UI/footer/footer";
import { useLocale } from "../context/LocaleContext";
import { apiJSON } from "../lib/api";

interface Preview {
  original_url: string;
  short_code: string;
  click_count: number;
}

export default function DemoLink() {
  const { code } = useParams();
  const { locale } = useLocale();
  const ru = locale === "ru";
  const [link, setLink] = useState<Preview | null>(null);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const endpoint = `/api/demo/links/${encodeURIComponent(code || "")}`;

  useEffect(() => {
    const controller = new AbortController();
    setLink(null);
    setError("");
    apiJSON<Preview>(endpoint, { signal: controller.signal })
      .then(setLink)
      .catch((error) => {
        if (!controller.signal.aborted) setError(error.message);
      });
    return () => controller.abort();
  }, [endpoint, locale]);

  const visit = async () => {
    setBusy(true);
    try {
      setLink(await apiJSON<Preview>(`${endpoint}/visit`, { method: "POST" }));
    } catch (error) {
      setError(error instanceof Error ? error.message : "Demo error");
      setLink(null);
    } finally {
      setBusy(false);
    }
  };

  const safeDestination = link && /^https?:\/\//i.test(link.original_url);
  return (
    <>
      <Header />
      <main className="demo-preview">
        <span>SHORTLI / SANDBOX</span>
        <h1>{ru ? "Переход без сюрпризов." : "A link, without surprises."}</h1>
        <p>
          {ru
            ? "В полной версии здесь происходит редирект. В демо мы сначала показываем адрес назначения — переходите только на знакомые вам сайты."
            : "The full version redirects here. In this demo, we show the destination first. Only open websites you trust."}
        </p>
        {error ? (
          <p role="alert">{error}</p>
        ) : link ? (
          <section>
            <code>{link.original_url}</code>
            <p role="status">
              {ru ? "Учебных переходов" : "Demo visits"}: {link.click_count}
            </p>
            <div className="demo-preview-actions">
              <button
                type="button"
                disabled={busy}
                onClick={() => void visit()}
              >
                {ru ? "Добавить учебный переход" : "Simulate a visit"}
              </button>
              {safeDestination && (
                <a
                  href={link.original_url}
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  {ru ? "Открыть внешний сайт ↗" : "Open external website ↗"}
                </a>
              )}
            </div>
            <p>
              {ru
                ? "Учебный переход обновляет локальную аналитику без открытия сайта. Внешняя ссылка открывается отдельно и не учитывается автоматически."
                : "Simulating a visit updates local analytics without opening a website. The external link opens separately and is not counted automatically."}
            </p>
          </section>
        ) : (
          <p>{ru ? "Загрузка…" : "Loading…"}</p>
        )}
        <Link to="/">
          {ru ? "← Вернуться в песочницу" : "← Back to the sandbox"}
        </Link>
      </main>
      <Footer />
    </>
  );
}
