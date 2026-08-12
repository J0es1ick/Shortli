import { useState } from "react";
import { buildShortUrl } from "../../lib/urls";
import type { URL } from "./urlList";
import styles from "./urlItem.module.css";
import { useLocale } from "../../context/LocaleContext";

interface URLItemProps {
  number: number;
  url: URL;
  onDelete: (shortCode: string) => Promise<void>;
  onManage: () => void;
}

export default function URLItem({
  number,
  url,
  onDelete,
  onManage,
}: URLItemProps) {
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const shortUrl = buildShortUrl(url.short_code);
  const { t, formatDate, formatNumber } = useLocale();

  return (
    <article className={styles.url_item}>
      <span>{String(number).padStart(2, "0")}</span>
      <a
        href={url.original_url}
        target="_blank"
        rel="noopener noreferrer"
        title={url.original_url}
      >
        {url.original_url}
      </a>
      <strong>{url.short_code}</strong>
      <span>{url.user_id ?? t("admin.guest")}</span>
      <span>{formatNumber(url.click_count ?? 0)}</span>
      <time dateTime={url.created_at}>{formatDate(url.created_at)}</time>
      <a
        href={shortUrl}
        target="_blank"
        rel="noopener noreferrer"
        title={shortUrl}
      >
        {shortUrl.replace(/^https?:\/\//, "")}
      </a>
      <div className={styles.actions}>
        <button
          type="button"
          className={styles.manage_button}
          onClick={onManage}
          aria-label={t("history.manage")}
        >
          ↗
        </button>
        {confirmDelete ? (
          <div className={styles.delete_confirm}>
            <button
              type="button"
              onClick={async () => {
                setDeleting(true);
                await onDelete(url.short_code);
                setDeleting(false);
              }}
              disabled={deleting}
            >
              {deleting ? "…" : t("common.yes")}
            </button>
            <button type="button" onClick={() => setConfirmDelete(false)}>
              {t("common.no")}
            </button>
          </div>
        ) : (
          <button
            type="button"
            className={styles.delete_button}
            onClick={() => setConfirmDelete(true)}
            aria-label={t("admin.deleteAria", { code: url.short_code })}
          >
            ×
          </button>
        )}
      </div>
    </article>
  );
}
