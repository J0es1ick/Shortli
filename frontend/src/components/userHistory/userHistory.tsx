import { useCallback, useEffect, useState } from "react";
import { useUser } from "../../context/UserContext";
import { apiUrl, buildShortUrl } from "../../lib/urls";
import ShareModal from "../UI/shareModal/shareModal";
import LinkDetailsModal, {
  type ManagedLink,
} from "../UI/linkDetailsModal/linkDetailsModal";
import styles from "./userHistory.module.css";
import { useLocale } from "../../context/LocaleContext";

interface ShortURL extends ManagedLink {
  qr_code_base64: string;
}

interface HistoryResponse {
  data: ShortURL[];
  meta: {
    total: number;
    page: number;
    limit: number;
    totalPages: number;
  };
}

const limit = 8;

export default function UserHistory() {
  const { user } = useUser();
  const { locale, t, formatDate, formatNumber } = useLocale();
  const [urls, setUrls] = useState<ShortURL[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [page, setPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const [copiedCode, setCopiedCode] = useState("");
  const [pendingDelete, setPendingDelete] = useState("");
  const [deletingCode, setDeletingCode] = useState("");
  const [shareItem, setShareItem] = useState<ShortURL | null>(null);
  const [detailsItem, setDetailsItem] = useState<ShortURL | null>(null);

  const fetchUserHistory = useCallback(async () => {
    if (!user) return;
    setLoading(true);
    setError("");

    try {
      const response = await fetch(
        apiUrl(`/api/history?page=${page}&limit=${limit}`),
        { credentials: "include" },
      );
      if (!response.ok) throw new Error(t("history.loadError"));

      const data: HistoryResponse = await response.json();
      setUrls(data.data);
      setTotalPages(Math.max(1, data.meta.totalPages));
    } catch (requestError) {
      setError(
        requestError instanceof Error
          ? requestError.message
          : t("history.loadError"),
      );
    } finally {
      setLoading(false);
    }
  }, [page, t, user]);

  useEffect(() => {
    void fetchUserHistory();
  }, [fetchUserHistory]);

  useEffect(() => {
    const refresh = () => {
      if (page === 1) void fetchUserHistory();
      else setPage(1);
    };
    window.addEventListener("shortli:url-created", refresh);
    return () => window.removeEventListener("shortli:url-created", refresh);
  }, [fetchUserHistory, page]);

  const copyLink = async (item: ShortURL) => {
    try {
      await navigator.clipboard.writeText(
        buildShortUrl(item.short_code, item.short_url),
      );
      setCopiedCode(item.short_code);
      window.setTimeout(() => setCopiedCode(""), 1600);
    } catch {
      setError(t("history.clipboardUnavailable"));
    }
  };

  const deleteLink = async (shortCode: string) => {
    setDeletingCode(shortCode);
    setError("");
    try {
      const response = await fetch(apiUrl(`/api/urls/${shortCode}`), {
        method: "DELETE",
        credentials: "include",
      });
      if (!response.ok) throw new Error(t("history.deleteError"));
      setPendingDelete("");
      if (urls.length === 1 && page > 1) setPage((current) => current - 1);
      else await fetchUserHistory();
    } catch (requestError) {
      setError(
        requestError instanceof Error
          ? requestError.message
          : t("history.deleteError"),
      );
    } finally {
      setDeletingCode("");
    }
  };

  const getExpiration = (item: ShortURL) => {
    if (!item.is_active) return { label: t("history.paused"), urgent: true };
    if (!item.expires_at)
      return { label: t("history.neverExpires"), urgent: false };
    const deletionDate = new Date(item.expires_at);
    const days = Math.ceil(
      (deletionDate.getTime() - Date.now()) / (1000 * 60 * 60 * 24),
    );
    if (days <= 0) return { label: t("history.expired"), urgent: true };
    if (days === 1) return { label: t("history.oneDayLeft"), urgent: true };
    if (days <= 7)
      return { label: t("history.daysLeft", { count: days }), urgent: true };
    return { label: t("history.daysLeft", { count: days }), urgent: false };
  };

  const getClickLabel = (count: number) => {
    if (locale === "en")
      return count === 1 ? t("history.click") : t("history.clicks");
    const lastTwoDigits = count % 100;
    const lastDigit = count % 10;
    if (lastTwoDigits >= 11 && lastTwoDigits <= 14) return t("history.clicks");
    if (lastDigit === 1) return t("history.click");
    if (lastDigit >= 2 && lastDigit <= 4) return t("history.clickFew");
    return t("history.clicks");
  };

  if (loading) {
    return (
      <div className={styles.loading} aria-live="polite">
        <i />
        <span>{t("history.loading")}</span>
      </div>
    );
  }

  if (error && urls.length === 0) {
    return (
      <div className={styles.error_state}>
        <p>{error}</p>
        <button type="button" onClick={() => void fetchUserHistory()}>
          {t("common.tryAgain")}
        </button>
      </div>
    );
  }

  if (urls.length === 0) {
    return (
      <div className={styles.empty_state}>
        <span>{t("history.emptyLabel")}</span>
        <p>{t("history.empty")}</p>
      </div>
    );
  }

  return (
    <div className={styles.user_history}>
      {error && (
        <p className={styles.inline_error} role="alert">
          {error}
        </p>
      )}
      <div className={styles.history_header} aria-hidden="true">
        <span>{t("history.link")}</span>
        <span>{t("history.activity")}</span>
        <span>{t("history.actions")}</span>
      </div>
      <div className={styles.history_list}>
        {urls.map((item, index) => {
          const shortUrl = buildShortUrl(item.short_code, item.short_url);
          const expiration = getExpiration(item);
          return (
            <article key={item.url_id} className={styles.history_item}>
              <div className={styles.item_index}>
                {String((page - 1) * limit + index + 1).padStart(2, "0")}
              </div>
              <div className={styles.link_info}>
                <a href={shortUrl} target="_blank" rel="noopener noreferrer">
                  {shortUrl.replace(/^https?:\/\//, "")}
                </a>
                <a
                  href={item.original_url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className={styles.original_url}
                  title={item.original_url}
                >
                  {item.original_url}
                </a>
              </div>
              <div className={styles.activity}>
                <strong>{formatNumber(item.click_count ?? 0)}</strong>
                <span>{getClickLabel(item.click_count ?? 0)}</span>
              </div>
              <div className={styles.date_info}>
                <time dateTime={item.created_at}>
                  {formatDate(item.created_at, {
                    day: "2-digit",
                    month: "short",
                    year: "numeric",
                  })}
                </time>
                <span className={expiration.urgent ? styles.urgent : ""}>
                  {expiration.label}
                </span>
              </div>
              <div className={styles.item_actions}>
                {pendingDelete === item.short_code ? (
                  <div className={styles.delete_confirm}>
                    <button
                      type="button"
                      onClick={() => void deleteLink(item.short_code)}
                      disabled={deletingCode === item.short_code}
                    >
                      {deletingCode === item.short_code
                        ? t("history.deleting")
                        : t("common.confirm")}
                    </button>
                    <button type="button" onClick={() => setPendingDelete("")}>
                      {t("common.cancel")}
                    </button>
                  </div>
                ) : (
                  <>
                    <button type="button" onClick={() => void copyLink(item)}>
                      {copiedCode === item.short_code
                        ? t("common.copied")
                        : t("common.copy")}
                    </button>
                    <button type="button" onClick={() => setShareItem(item)}>
                      {t("common.share")}
                    </button>
                    <button type="button" onClick={() => setDetailsItem(item)}>
                      {t("history.manage")}
                    </button>
                    <button
                      type="button"
                      className={styles.delete_button}
                      onClick={() => setPendingDelete(item.short_code)}
                    >
                      {t("common.delete")}
                    </button>
                  </>
                )}
              </div>
            </article>
          );
        })}
      </div>

      {totalPages > 1 && (
        <nav className={styles.pagination} aria-label={t("history.pages")}>
          <button
            type="button"
            onClick={() => setPage((current) => Math.max(current - 1, 1))}
            disabled={page === 1}
          >
            {t("history.previous")}
          </button>
          <span>
            {String(page).padStart(2, "0")} /{" "}
            {String(totalPages).padStart(2, "0")}
          </span>
          <button
            type="button"
            onClick={() =>
              setPage((current) => Math.min(current + 1, totalPages))
            }
            disabled={page === totalPages}
          >
            {t("history.next")}
          </button>
        </nav>
      )}

      <ShareModal
        isOpen={Boolean(shareItem)}
        onClose={() => setShareItem(null)}
        url={
          shareItem
            ? buildShortUrl(shareItem.short_code, shareItem.short_url)
            : ""
        }
      />
      <LinkDetailsModal
        item={detailsItem}
        onClose={() => setDetailsItem(null)}
        onUpdated={async () => {
          await fetchUserHistory();
          setDetailsItem(null);
        }}
      />
    </div>
  );
}
