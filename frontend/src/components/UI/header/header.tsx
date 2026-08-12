import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { useTheme } from "../../../context/ThemeContext";
import { useUser } from "../../../context/UserContext";
import { useLocale } from "../../../context/LocaleContext";
import LoginModal from "../loginModal/loginModal";
import styles from "./header.module.css";

export function Header() {
  const { toggleTheme, theme } = useTheme();
  const { user, logout, isLoading } = useUser();
  const { locale, toggleLocale, t } = useLocale();
  const [isLoginOpen, setIsLoginOpen] = useState(false);
  const [isLogoutOpen, setIsLogoutOpen] = useState(false);
  const [isLoggingOut, setIsLoggingOut] = useState(false);

  useEffect(() => {
    if (!isLogoutOpen) return;
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") setIsLogoutOpen(false);
    };
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [isLogoutOpen]);

  const confirmLogout = async () => {
    setIsLoggingOut(true);
    await logout();
    setIsLoggingOut(false);
    setIsLogoutOpen(false);
  };

  return (
    <>
      <header className={styles.header}>
        <Link to="/" className={styles.brand} aria-label={t("header.home")}>
          <span className={styles.brand_mark} aria-hidden="true">
            <i />
          </span>
          <span>Shortli</span>
        </Link>

        <nav className={styles.navigation} aria-label={t("header.navigation")}>
          <a href="/#shorten">{t("header.shorten")}</a>
          <a href="/#history">{t("header.archive")}</a>
          <Link to="/developers">{t("header.api")}</Link>
          {user?.is_admin && <Link to="/stats">{t("header.stats")}</Link>}
        </nav>

        <div className={styles.header_actions}>
          <button
            type="button"
            className={styles.theme_toggle}
            onClick={toggleTheme}
            aria-label={t("header.switchTheme", {
              theme: theme === "dark" ? t("header.light") : t("header.dark"),
            })}
          >
            <span>
              {theme === "dark" ? t("header.dark") : t("header.light")}
            </span>
            <i
              className={theme === "dark" ? styles.dark : ""}
              aria-hidden="true"
            />
          </button>

          <button
            type="button"
            className={styles.locale_toggle}
            onClick={toggleLocale}
            aria-label={t("header.switchLanguage", {
              language:
                locale === "ru" ? t("header.english") : t("header.russian"),
            })}
          >
            <i
              className={locale === "ru" ? styles.russian : ""}
              aria-hidden="true"
            />
            <span className={locale === "en" ? styles.active_locale : ""}>
              EN
            </span>
            <span className={locale === "ru" ? styles.active_locale : ""}>
              RU
            </span>
          </button>

          {isLoading ? (
            <span className={styles.auth_loading}>{t("header.sync")}</span>
          ) : user ? (
            <button
              type="button"
              className={styles.account_button}
              onClick={() => setIsLogoutOpen(true)}
              title={user.email}
            >
              <span>{user.email.split("@")[0]}</span>
              <span>{t("header.logOut")}</span>
            </button>
          ) : (
            <button
              type="button"
              className={styles.sign_in_button}
              onClick={() => setIsLoginOpen(true)}
            >
              {t("header.signIn")}
            </button>
          )}
        </div>
      </header>

      <LoginModal isOpen={isLoginOpen} onClose={() => setIsLoginOpen(false)} />

      {isLogoutOpen && (
        <div
          className={styles.confirm_overlay}
          onMouseDown={(event) => {
            if (event.target === event.currentTarget) setIsLogoutOpen(false);
          }}
        >
          <div
            className={styles.confirm_dialog}
            role="alertdialog"
            aria-modal="true"
            aria-labelledby="logout-title"
            aria-describedby="logout-description"
          >
            <span>{t("header.sessionEnd")}</span>
            <h2 id="logout-title">{t("header.logoutTitle")}</h2>
            <p id="logout-description">{t("header.logoutDescription")}</p>
            <div>
              <button type="button" onClick={() => setIsLogoutOpen(false)}>
                {t("header.staySignedIn")}
              </button>
              <button
                type="button"
                onClick={confirmLogout}
                disabled={isLoggingOut}
              >
                {isLoggingOut ? t("header.signingOut") : t("header.logOut")}
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  );
}
