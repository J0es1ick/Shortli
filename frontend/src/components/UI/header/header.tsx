import { useState } from "react";
import { createPortal } from "react-dom";
import { useDialog } from "../../../hooks/useDialog";
import { Link } from "react-router-dom";
import { useTheme } from "../../../context/ThemeContext";
import { useUser } from "../../../context/UserContext";
import { hasStaffAccess } from "../../../lib/userAccess";
import { useLocale } from "../../../context/LocaleContext";
import LoginModal from "../loginModal/loginModal";
import styles from "./header.module.css";

export function Header() {
  const { toggleTheme, theme } = useTheme();
  const { user, logout, isLoading, authError, checkAuth } = useUser();
  const { locale, toggleLocale, t, apiError } = useLocale();
  const [isLoginOpen, setIsLoginOpen] = useState(false);
  const [isLogoutOpen, setIsLogoutOpen] = useState(false);
  const [isLoggingOut, setIsLoggingOut] = useState(false);

  const [logoutError, setLogoutError] = useState("");
  const closeLogout = () => {
    if (!isLoggingOut) setIsLogoutOpen(false);
  };
  const dialogRef = useDialog(isLogoutOpen, closeLogout);

  const confirmLogout = async () => {
    setIsLoggingOut(true);
    setLogoutError("");
    try {
      await logout();
      setIsLogoutOpen(false);
    } catch (error) {
      setLogoutError(
        apiError(
          error instanceof Error ? error.message : undefined,
          "header.logoutFailed",
        ),
      );
    } finally {
      setIsLoggingOut(false);
    }
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
          {hasStaffAccess(user) && (
            <Link to="/stats">{t("header.control")}</Link>
          )}
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
          ) : authError ? (
            <button
              type="button"
              className={styles.sign_in_button}
              onClick={() => void checkAuth()}
              title={t("header.authFailed")}
            >
              {t("header.retryAuth")}
            </button>
          ) : user ? (
            <button
              type="button"
              className={styles.account_button}
              onClick={() => {
                setLogoutError("");
                setIsLogoutOpen(true);
              }}
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

      {isLogoutOpen &&
        createPortal(
          <div
            className={styles.confirm_overlay}
            onMouseDown={(event) => {
              if (event.target === event.currentTarget) closeLogout();
            }}
          >
            <section
              ref={dialogRef}
              tabIndex={-1}
              className={styles.confirm_dialog}
              role="alertdialog"
              aria-modal="true"
              aria-labelledby="logout-title"
              aria-describedby="logout-description"
            >
              <span>{t("header.sessionEnd")}</span>
              <h2 id="logout-title">{t("header.logoutTitle")}</h2>
              <p id="logout-description">{t("header.logoutDescription")}</p>
              {logoutError && <p role="alert">{logoutError}</p>}
              <div>
                <button
                  type="button"
                  onClick={closeLogout}
                  disabled={isLoggingOut}
                >
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
            </section>
          </div>,
          document.body,
        )}
    </>
  );
}
