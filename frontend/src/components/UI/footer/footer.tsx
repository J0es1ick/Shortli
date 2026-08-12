import styles from "./footer.module.css";
import { useLocale } from "../../../context/LocaleContext";
import { Link } from "react-router-dom";

export default function Footer() {
  const { t } = useLocale();
  return (
    <footer className={styles.footer}>
      <div className={styles.footer_inner}>
        <div className={styles.brand}>
          <span>SHORTLI</span>
          <p>{t("footer.tagline")}</p>
        </div>
        <div className={styles.footer_meta}>
          <span>{t("footer.built")}</span>
          <span>© {new Date().getFullYear()}</span>
        </div>
        <nav aria-label={t("footer.navigation")}>
          <a
            href="https://t.me/Joes1ick"
            target="_blank"
            rel="noopener noreferrer"
          >
            {t("footer.contact")}
          </a>
          <a href="/#shorten">{t("footer.create")}</a>
          <a href="/#history">{t("footer.archive")}</a>
          <Link to="/privacy">{t("footer.privacy")}</Link>
          <Link to="/terms">{t("footer.terms")}</Link>
          <Link to="/status">{t("footer.status")}</Link>
          <Link to="/report">{t("footer.report")}</Link>
        </nav>
      </div>
    </footer>
  );
}
