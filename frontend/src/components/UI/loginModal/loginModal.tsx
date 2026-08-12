import { useEffect, useRef, useState, type FormEvent } from "react";
import { createPortal } from "react-dom";
import { Link } from "react-router-dom";
import { useUser } from "../../../context/UserContext";
import { apiUrl } from "../../../lib/urls";
import styles from "./loginModal.module.css";
import { useLocale } from "../../../context/LocaleContext";

interface LoginResponse {
  user_id: number;
  email: string;
  is_admin: boolean;
  created_at: string;
}

interface LoginModalProps {
  isOpen: boolean;
  onClose: () => void;
  onLoginSuccess?: () => void;
}

export default function LoginModal({
  isOpen,
  onClose,
  onLoginSuccess,
}: LoginModalProps) {
  const { login } = useUser();
  const { t, apiError } = useLocale();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const emailRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (!isOpen) return;
    const previousOverflow = document.body.style.overflow;
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") onClose();
    };
    document.body.style.overflow = "hidden";
    document.addEventListener("keydown", handleKeyDown);
    window.setTimeout(() => emailRef.current?.focus(), 80);

    return () => {
      document.body.style.overflow = previousOverflow;
      document.removeEventListener("keydown", handleKeyDown);
    };
  }, [isOpen, onClose]);

  const closeModal = () => {
    setError("");
    setSuccess("");
    onClose();
  };

  const handleLogin = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setLoading(true);
    setError("");
    setSuccess("");

    try {
      const response = await fetch(apiUrl("/api/login"), {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({ email, password }),
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(apiError(errorData.error, "login.failed"));
      }

      const data: LoginResponse = await response.json();
      login(data);
      setSuccess(t("login.welcome", { email: data.email }));
      setEmail("");
      setPassword("");
      window.setTimeout(() => {
        onLoginSuccess?.();
        onClose();
        setSuccess("");
      }, 650);
    } catch (requestError) {
      setError(
        requestError instanceof Error
          ? requestError.message
          : t("login.genericError"),
      );
    } finally {
      setLoading(false);
    }
  };

  if (!isOpen) return null;

  return createPortal(
    <div
      className={styles.modal_overlay}
      onMouseDown={(event) => {
        if (event.target === event.currentTarget) closeModal();
      }}
    >
      <div
        className={styles.signin_modal}
        role="dialog"
        aria-modal="true"
        aria-labelledby="signin-title"
      >
        <div className={styles.modal_header}>
          <div>
            <span>{t("login.access")}</span>
            <h2 id="signin-title">{t("login.title")}</h2>
          </div>
          <button
            onClick={closeModal}
            className={styles.close_button}
            aria-label={t("login.close")}
          >
            ×
          </button>
        </div>
        <form onSubmit={handleLogin}>
          <label className={styles.form_group}>
            <span>{t("login.email")}</span>
            <input
              ref={emailRef}
              type="email"
              value={email}
              onChange={(event) => setEmail(event.target.value)}
              placeholder={t("login.emailPlaceholder")}
              autoComplete="email"
              required
              disabled={loading}
            />
          </label>
          <label className={styles.form_group}>
            <span>{t("login.password")}</span>
            <input
              type="password"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              placeholder={t("login.passwordPlaceholder")}
              autoComplete="current-password"
              required
              disabled={loading}
            />
          </label>
          <button
            type="submit"
            className={styles.submit_button}
            disabled={loading}
          >
            <span>{loading ? t("login.signingIn") : t("header.signIn")}</span>
            <span aria-hidden="true">↗</span>
          </button>

          {error && (
            <div className={styles.error_message} role="alert">
              {error}
            </div>
          )}
          {success && (
            <div className={styles.success_message} role="status">
              {success}
            </div>
          )}

          <p className={styles.register}>
            {t("login.new")}{" "}
            <Link to="/register" onClick={closeModal}>
              {t("login.create")}
            </Link>
          </p>
        </form>
      </div>
    </div>,
    document.body,
  );
}
