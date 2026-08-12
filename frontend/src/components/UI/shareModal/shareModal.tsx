import { useEffect, useState, type CSSProperties } from "react";
import { createPortal } from "react-dom";
import styles from "./shareModal.module.css";
import { useLocale } from "../../../context/LocaleContext";

interface ShareModalProps {
  isOpen: boolean;
  onClose: () => void;
  url: string;
  title?: string;
  description?: string;
}

interface SharePlatform {
  name: string;
  mark: string;
  color: string;
  url: string;
}

export default function ShareModal({
  isOpen,
  onClose,
  url,
  title,
  description,
}: ShareModalProps) {
  const { t } = useLocale();
  const [copied, setCopied] = useState("");
  const nativeShare = (
    navigator as unknown as {
      share?: (data: ShareData) => Promise<void>;
    }
  ).share;

  useEffect(() => {
    if (!isOpen) return;
    const previousOverflow = document.body.style.overflow;
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") onClose();
    };
    document.body.style.overflow = "hidden";
    document.addEventListener("keydown", handleKeyDown);

    return () => {
      document.body.style.overflow = previousOverflow;
      document.removeEventListener("keydown", handleKeyDown);
    };
  }, [isOpen, onClose]);

  if (!isOpen) return null;

  const resolvedTitle = title || t("shortener.shareTitle");
  const resolvedDescription =
    description || t("shortener.shareDescription", { url });
  const encodedUrl = encodeURIComponent(url);
  const encodedTitle = encodeURIComponent(resolvedTitle);
  const encodedDescription = encodeURIComponent(resolvedDescription);
  const htmlCode = `<a href="${url}" target="_blank" rel="noopener noreferrer">${resolvedTitle}</a>`;
  const markdownCode = `[${resolvedTitle}](${url})`;

  const sharePlatforms: SharePlatform[] = [
    {
      name: "X / Twitter",
      mark: "X",
      color: "#151515",
      url: `https://twitter.com/intent/tweet?text=${encodedTitle}&url=${encodedUrl}`,
    },
    {
      name: "LinkedIn",
      mark: "in",
      color: "#0a66c2",
      url: `https://www.linkedin.com/shareArticle?mini=true&url=${encodedUrl}&title=${encodedTitle}&summary=${encodedDescription}`,
    },
    {
      name: "Telegram",
      mark: "TG",
      color: "#168acd",
      url: `https://t.me/share/url?url=${encodedUrl}&text=${encodedTitle}`,
    },
    {
      name: "WhatsApp",
      mark: "WA",
      color: "#1f9e59",
      url: `https://wa.me/?text=${encodedTitle}%20${encodedUrl}`,
    },
    {
      name: "Facebook",
      mark: "f",
      color: "#1877f2",
      url: `https://www.facebook.com/sharer/sharer.php?u=${encodedUrl}`,
    },
    {
      name: "Reddit",
      mark: "r/",
      color: "#e84b18",
      url: `https://reddit.com/submit?url=${encodedUrl}&title=${encodedTitle}`,
    },
  ];

  const copyText = async (value: string, key: string) => {
    try {
      await navigator.clipboard.writeText(value);
      setCopied(key);
      window.setTimeout(() => setCopied(""), 1600);
    } catch {
      setCopied("error");
    }
  };

  const handleNativeShare = async () => {
    if (!nativeShare) return;
    try {
      await nativeShare.call(navigator, {
        title: resolvedTitle,
        text: resolvedDescription,
        url,
      });
    } catch (shareError) {
      if ((shareError as Error).name !== "AbortError") setCopied("error");
    }
  };

  const openShareWindow = (platform: SharePlatform) => {
    const width = 640;
    const height = 520;
    window.open(
      platform.url,
      "shortli-share",
      `noopener,noreferrer,width=${width},height=${height},left=${Math.max(
        0,
        (window.innerWidth - width) / 2,
      )},top=${Math.max(0, (window.innerHeight - height) / 2)}`,
    );
  };

  return createPortal(
    <div
      className={styles.modal_overlay}
      onMouseDown={(event) => {
        if (event.target === event.currentTarget) onClose();
      }}
    >
      <div
        className={styles.modal}
        role="dialog"
        aria-modal="true"
        aria-labelledby="share-modal-title"
      >
        <header className={styles.modal_header}>
          <div>
            <span>{t("share.distribute")}</span>
            <h2 id="share-modal-title">{t("share.title")}</h2>
          </div>
          <button
            onClick={onClose}
            className={styles.close_button}
            aria-label={t("share.close")}
          >
            ×
          </button>
        </header>

        <div className={styles.url_preview}>
          <div>
            <span>{t("share.shortUrl")}</span>
            <strong>{url.replace(/^https?:\/\//, "")}</strong>
          </div>
          <button type="button" onClick={() => copyText(url, "url")}>
            {copied === "url" ? t("common.copied") : t("share.copyLink")}
          </button>
        </div>

        {nativeShare && (
          <button
            type="button"
            onClick={handleNativeShare}
            className={styles.native_share_button}
          >
            <span>{t("share.device")}</span>
            <span aria-hidden="true">↗</span>
          </button>
        )}

        <section
          className={styles.platforms}
          aria-labelledby="share-platforms-heading"
        >
          <div className={styles.section_heading}>
            <span>01</span>
            <h3 id="share-platforms-heading">{t("share.channel")}</h3>
          </div>
          <div className={styles.platforms_grid}>
            {sharePlatforms.map((platform) => (
              <button
                type="button"
                key={platform.name}
                onClick={() => openShareWindow(platform)}
                className={styles.platform_button}
                style={{ "--platform-color": platform.color } as CSSProperties}
              >
                <span className={styles.platform_mark}>{platform.mark}</span>
                <span>{platform.name}</span>
                <span aria-hidden="true">↗</span>
              </button>
            ))}
          </div>
        </section>

        <section
          className={styles.embed_section}
          aria-labelledby="embed-heading"
        >
          <div className={styles.section_heading}>
            <span>02</span>
            <h3 id="embed-heading">{t("share.code")}</h3>
          </div>
          <div className={styles.embed_options}>
            <div className={styles.embed_option}>
              <span>HTML</span>
              <code>{htmlCode}</code>
              <button type="button" onClick={() => copyText(htmlCode, "html")}>
                {copied === "html" ? t("common.copied") : t("share.copyHtml")}
              </button>
            </div>
            <div className={styles.embed_option}>
              <span>MARKDOWN</span>
              <code>{markdownCode}</code>
              <button
                type="button"
                onClick={() => copyText(markdownCode, "markdown")}
              >
                {copied === "markdown"
                  ? t("common.copied")
                  : t("share.copyMarkdown")}
              </button>
            </div>
          </div>
        </section>

        {copied === "error" && (
          <p className={styles.error} role="alert">
            {t("share.clipboardUnavailable")}
          </p>
        )}
      </div>
    </div>,
    document.body,
  );
}
