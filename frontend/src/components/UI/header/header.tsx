import styles from "./header.module.css";
import { useTheme } from "../../../context/ThemeContext.tsx";
import { useState } from "react";
import { useUser } from "../../../context/UserContext.tsx";
import LoginModal from "../loginModal/loginModal";

export function Header() {
  const { toggleTheme, theme } = useTheme();
  const { user, logout, isLoading } = useUser();
  const [isModalOpen, setIsModalOpen] = useState(false);

  const handleSignInClick = () => {
    setIsModalOpen(true);
  };

  const handleLogout = async () => {
    await logout();
  };

  const shouldShowStats = user && user.is_admin;

  if (isLoading) {
    return (
      <header className={styles.header}>
        <div className={styles.header_left}>
          <img src="public/logo.svg" alt="logo" />
          <a href="/" target="_parent">
            Shortli
          </a>
        </div>
        <div className={styles.header_right}>
          <div className={styles.theme_toggle} onClick={toggleTheme}>
            <img className={styles.sun} src="public/brightness.svg" alt="sun" />
            <img className={styles.moon} src="public/moon.svg" alt="moon" />
          </div>
          <button className={styles.sign_in_button} disabled>
            Loading...
          </button>
        </div>
      </header>
    );
  }

  return (
    <>
      <header className={styles.header}>
        <div className={styles.header_left}>
          <img src="public/logo.svg" alt="logo" />
          <a href="/" target="_parent">
            Shortli
          </a>
        </div>
        <div className={styles.header_center}>
          {shouldShowStats && (
            <ul>
              <li>
                <a href="/stats">Stats</a>
              </li>
            </ul>
          )}
        </div>
        <div className={styles.header_right}>
          <div
            className={`${styles.theme_toggle} ${
              theme === "dark" ? styles.dark : ""
            }`}
            onClick={toggleTheme}
          >
            <img className={styles.sun} src="public/brightness.svg" alt="sun" />
            <img className={styles.moon} src="public/moon.svg" alt="moon" />
          </div>
          {user ? (
            <button className={styles.logout_button} onClick={handleLogout}>
              Logout
            </button>
          ) : (
            <button
              className={styles.sign_in_button}
              onClick={handleSignInClick}
            >
              Sign in
            </button>
          )}
        </div>
      </header>
      <LoginModal isOpen={isModalOpen} onClose={() => setIsModalOpen(false)} />
    </>
  );
}
