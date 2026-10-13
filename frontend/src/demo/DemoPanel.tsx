import { useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { useLocale } from "../context/LocaleContext";
import { useUser } from "../context/UserContext";
import { apiJSON } from "../lib/api";
import "./demo.css";

export default function DemoPanel({ onReset }: { onReset: () => void }) {
  const { locale, t } = useLocale();
  const { user, isLoading, checkAuth } = useUser();
  const navigate = useNavigate();
  const [busy, setBusy] = useState(false);
  const [confirm, setConfirm] = useState(false);
  const [error, setError] = useState("");
  const ru = locale === "ru";

  const change = async (role?: string) => {
    setBusy(true);
    setError("");
    try {
      await apiJSON(role ? "/api/demo/role" : "/api/demo/reset", {
        method: "POST",
        body: JSON.stringify({ role }),
      });
      await checkAuth();
      onReset();
      navigate(role && !["guest", "user"].includes(role) ? "/stats" : "/");
      setConfirm(false);
    } catch (error) {
      setError(error instanceof Error ? error.message : "Demo error");
    } finally {
      setBusy(false);
    }
  };

  return (
    <aside className="demo-panel" aria-label={ru ? "Песочница" : "Sandbox"}>
      <div className="demo-panel-heading">
        <strong>SHORTLI / DEMO</strong>
        <span>
          {ru
            ? "Без сервера · учебная статистика · данные только в вашем браузере"
            : "No server · sample analytics · data stays in your browser"}
        </span>
      </div>
      <div className="demo-controls">
        <label htmlFor="demo-role">
          {ru ? "Попробовать роль" : "Try a role"}
        </label>
        <select
          id="demo-role"
          value={user?.role || "guest"}
          disabled={busy || isLoading}
          onChange={(event) => void change(event.target.value)}
        >
          <option value="guest">{ru ? "Гость" : "Guest"}</option>
          {(["user", "support", "admin", "owner"] as const).map((role) => (
            <option key={role} value={role}>
              {t(`admin.role.${role}`)}
            </option>
          ))}
        </select>
        <button
          type="button"
          disabled={busy}
          onClick={() => setConfirm((value) => !value)}
        >
          {ru ? "Сбросить демо" : "Reset demo"}
        </button>
        <details>
          <summary>{ru ? "Как это работает" : "How it works"}</summary>
          <div className="demo-explanation">
            <p>
              {ru
                ? "Создайте ссылку и QR, откройте архив, посмотрите аналитику. Выберите владельца для управления пользователями, администратора или техподдержку — для модерации."
                : "Create a link and QR, explore your archive and analytics. Choose Owner to manage users, or Admin and Support to try moderation."}
            </p>
            <p>
              {ru
                ? "Нет сервера, настоящих аккаунтов, API-ключей и отправки писем. Статистика синтетическая; можно добавлять учебные переходы. Не используйте настоящие пароли, личные данные или секретные ссылки."
                : "There is no server, real accounts, API keys or email delivery. Analytics are synthetic; you can add demo visits. Do not use real passwords, personal data or secret links."}
            </p>
            <p>
              {ru
                ? "Новые ссылки и QR работают только в этом браузере и профиле. На другом устройстве их данных не будет. Сброс удалит ваши изменения и вернёт примеры."
                : "New links and QR codes work only in this browser profile. Another device will not have their data. Reset removes your changes and restores the examples."}
            </p>
            <Link to="/s/portfolio">
              {ru ? "Открыть пример перехода ↗" : "Open a sample link ↗"}
            </Link>
          </div>
        </details>
      </div>
      {confirm && (
        <div
          className="demo-confirm"
          role="group"
          aria-label={ru ? "Подтверждение сброса" : "Confirm reset"}
        >
          <span>
            {ru
              ? "Удалить локальные изменения и восстановить примеры?"
              : "Delete local changes and restore the examples?"}
          </span>
          <button type="button" disabled={busy} onClick={() => void change()}>
            {ru ? "Да, сбросить" : "Yes, reset"}
          </button>
          <button
            type="button"
            disabled={busy}
            onClick={() => setConfirm(false)}
          >
            {ru ? "Отмена" : "Cancel"}
          </button>
        </div>
      )}
      {error && <p role="alert">{error}</p>}
    </aside>
  );
}
