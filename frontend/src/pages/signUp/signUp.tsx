import { useState, type FormEvent } from "react";
import { Link, useNavigate } from "react-router-dom";
import NetworkMesh from "../../components/networkMesh/networkMesh";
import { Header } from "../../components/UI/header/header";
import { useUser } from "../../context/UserContext";
import { apiUrl } from "../../lib/urls";
import styles from "./signUp.module.css";
import { useLocale } from "../../context/LocaleContext";

export default function SignUpPage() {
  const navigate = useNavigate();
  const { login } = useUser();
  const { t, apiError } = useLocale();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const handleRegister = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setError("");

    if (password !== confirmPassword) {
      setError(t("signup.passwordMismatch"));
      return;
    }
    if (
      password.length < 10 ||
      !/[A-Za-zА-Яа-яЁё]/.test(password) ||
      !/[0-9]/.test(password)
    ) {
      setError(t("signup.passwordLength"));
      return;
    }

    setLoading(true);
    try {
      const registerResponse = await fetch(apiUrl("/api/register"), {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({ email, password }),
      });

      if (!registerResponse.ok) {
        const errorData = await registerResponse.json().catch(() => ({}));
        throw new Error(apiError(errorData.error, "signup.failed"));
      }

      const loginResponse = await fetch(apiUrl("/api/login"), {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({ email, password }),
      });
      if (!loginResponse.ok) throw new Error(t("signup.loginFailed"));

      login(await loginResponse.json());
      navigate("/");
    } catch (requestError) {
      setError(
        requestError instanceof Error
          ? requestError.message
          : t("signup.genericError"),
      );
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className={styles.page}>
      <Header />
      <main className={styles.main}>
        <section className={styles.intro} aria-labelledby="signup-title">
          <div>
            <span>{t("signup.members")}</span>
            <h1 id="signup-title">{t("signup.hero")}</h1>
            <p>{t("signup.intro")}</p>
          </div>
          <NetworkMesh />
        </section>

        <section
          className={styles.form_panel}
          aria-label={t("signup.createButton")}
        >
          <div className={styles.form_heading}>
            <span>{t("signup.create")}</span>
            <p>{t("signup.simple")}</p>
          </div>
          <form onSubmit={handleRegister}>
            <label>
              <span>{t("signup.email")}</span>
              <input
                type="email"
                value={email}
                onChange={(event) => setEmail(event.target.value)}
                placeholder={t("login.emailPlaceholder")}
                autoComplete="email"
                required
                disabled={loading}
              />
            </label>
            <label>
              <span>{t("signup.password")}</span>
              <input
                type="password"
                value={password}
                onChange={(event) => setPassword(event.target.value)}
                placeholder={t("signup.passwordPlaceholder")}
                autoComplete="new-password"
                minLength={10}
                maxLength={72}
                required
                disabled={loading}
              />
            </label>
            <label>
              <span>{t("signup.confirmPassword")}</span>
              <input
                type="password"
                value={confirmPassword}
                onChange={(event) => setConfirmPassword(event.target.value)}
                placeholder={t("signup.confirmPlaceholder")}
                autoComplete="new-password"
                required
                disabled={loading}
              />
            </label>

            <button type="submit" disabled={loading}>
              <span>
                {loading ? t("signup.creating") : t("signup.createButton")}
              </span>
              <span aria-hidden="true">↗</span>
            </button>

            {error && (
              <p className={styles.error} role="alert">
                {error}
              </p>
            )}
          </form>
          <p className={styles.signin_note}>
            {t("signup.member")}{" "}
            <Link to="/" state={{ openLoginModal: true }}>
              {t("signup.signIn")}
            </Link>
          </p>
        </section>
      </main>
    </div>
  );
}
