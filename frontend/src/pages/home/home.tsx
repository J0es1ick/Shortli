import { useEffect, useState } from "react";
import { Link, useLocation } from "react-router-dom";
import NetworkMesh from "../../components/networkMesh/networkMesh";
import ShortenerForm from "../../components/shortenerForm/shortenerForm";
import Footer from "../../components/UI/footer/footer";
import { Header } from "../../components/UI/header/header";
import LoginModal from "../../components/UI/loginModal/loginModal";
import UserHistory from "../../components/userHistory/userHistory";
import { useUser } from "../../context/UserContext";
import { useLocale } from "../../context/LocaleContext";
import styles from "./home.module.css";

export default function Home() {
  const { user, isLoading } = useUser();
  const { t } = useLocale();
  const [isLoginModalOpen, setIsLoginModalOpen] = useState(false);
  const location = useLocation();

  useEffect(() => {
    if (location.state?.openLoginModal) {
      setIsLoginModalOpen(true);
      window.history.replaceState({}, document.title);
    }
  }, [location]);

  return (
    <div className={styles.page}>
      <Header />
      <main className={styles.main}>
        <section className={styles.hero} aria-labelledby="hero-title">
          <div className={styles.hero_content}>
            <div className={styles.eyebrow}>
              <span>SHORTLI / 01</span>
              <span>{t("home.utility")}</span>
            </div>
            <h1 id="hero-title">
              {t("home.heroLine1")}
              <br />
              <span>{t("home.heroLine2")}</span>
            </h1>
            <p className={styles.intro}>{t("home.intro")}</p>
            <ShortenerForm />
            <div className={styles.hero_meta} aria-label={t("home.highlights")}>
              <span>{t("home.noNoise")}</span>
              <span>{t("home.instantQr")}</span>
              <span>{t("home.history30")}</span>
            </div>
          </div>
          <div className={styles.visual}>
            <NetworkMesh />
          </div>
        </section>

        <section
          className={styles.benefits_section}
          aria-labelledby="benefits-title"
        >
          <div className={styles.benefits_intro}>
            <span>{t("home.benefitsLabel")}</span>
            <h2 id="benefits-title">{t("home.benefitsTitle")}</h2>
            <p>{t("home.benefitsDescription")}</p>
            {!user && !isLoading && (
              <Link to="/register" className={styles.signup_button}>
                {t("home.createAccount")}
              </Link>
            )}
          </div>
          <div className={styles.benefit_grid}>
            {(["lifecycle", "analytics", "archive", "safety"] as const).map(
              (benefit, index) => (
                <article key={benefit}>
                  <span>{String(index + 1).padStart(2, "0")}</span>
                  <h3>{t(`home.benefit.${benefit}.title`)}</h3>
                  <p>{t(`home.benefit.${benefit}.description`)}</p>
                </article>
              ),
            )}
          </div>
          <div className={styles.trust_line} aria-label={t("home.trustLabel")}>
            <span>{t("home.trust.immutable")}</span>
            <span>{t("home.trust.private")}</span>
            <span>{t("home.trust.owner")}</span>
          </div>
        </section>

        <section className={styles.history_section} id="history">
          <div className={styles.section_heading}>
            <span>{t("home.archiveLabel")}</span>
            <h2>{t("home.archiveTitle")}</h2>
            <p>{t("home.archiveDescription")}</p>
          </div>

          {isLoading ? (
            <div className={styles.history_loading} aria-live="polite">
              {t("home.syncing")}
            </div>
          ) : user ? (
            <UserHistory />
          ) : (
            <div className={styles.history_prompt}>
              <div>
                <span className={styles.prompt_index}>
                  {t("home.membersArchive")}
                </span>
                <h3>{t("home.keepUseful")}</h3>
                <p>{t("home.accountDescription")}</p>
              </div>
              <div className={styles.prompt_actions}>
                <Link to="/register" className={styles.signup_button}>
                  {t("home.createAccount")}
                </Link>
                <button
                  type="button"
                  onClick={() => setIsLoginModalOpen(true)}
                  className={styles.signin_link}
                >
                  {t("header.signIn")}
                </button>
              </div>
            </div>
          )}
        </section>
      </main>
      <Footer />
      <LoginModal
        isOpen={isLoginModalOpen}
        onClose={() => setIsLoginModalOpen(false)}
      />
    </div>
  );
}
