import { useEffect } from "react";
import Footer from "../../components/UI/footer/footer";
import { Header } from "../../components/UI/header/header";
import { useLocale } from "../../context/LocaleContext";
import type { TranslationKey } from "../../i18n/translations";
import styles from "./infoPages.module.css";

export interface LegalSection {
  title: TranslationKey;
  paragraphs: TranslationKey[];
}

interface Props {
  label: TranslationKey;
  title: TranslationKey;
  intro: TranslationKey;
  pageTitle: TranslationKey;
  sections: LegalSection[];
}

export default function LegalPage({
  label,
  title,
  intro,
  pageTitle,
  sections,
}: Props) {
  const { t } = useLocale();

  useEffect(() => {
    document.title = `${t(pageTitle)} — Shortli`;
  }, [pageTitle, t]);

  return (
    <div className={styles.page}>
      <Header />
      <main className={styles.main}>
        <header className={styles.hero}>
          <span>{t(label)}</span>
          <h1>{t(title)}</h1>
          <p>{t(intro)}</p>
          <time dateTime="2026-07-23">{t("legal.updated")}</time>
        </header>
        <div className={styles.document}>
          <aside>
            <span>{t("legal.navigation")}</span>
            {sections.map((section, index) => (
              <a key={section.title} href={`#section-${index + 1}`}>
                {String(index + 1).padStart(2, "0")} / {t(section.title)}
              </a>
            ))}
          </aside>
          <article>
            {sections.map((section, index) => (
              <section key={section.title} id={`section-${index + 1}`}>
                <span>{String(index + 1).padStart(2, "0")}</span>
                <div>
                  <h2>{t(section.title)}</h2>
                  {section.paragraphs.map((paragraph) => (
                    <p key={paragraph}>{t(paragraph)}</p>
                  ))}
                </div>
              </section>
            ))}
          </article>
        </div>
      </main>
      <Footer />
    </div>
  );
}
