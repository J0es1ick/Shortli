import { useCallback, useEffect, useState, type FormEvent } from "react";
import { apiUrl } from "../../lib/urls";
import Pagination from "../UI/pagination/pagination";
import URLItem from "./urlItem";
import styles from "./urlList.module.css";
import { useLocale } from "../../context/LocaleContext";
import LinkDetailsModal from "../UI/linkDetailsModal/linkDetailsModal";

export interface URL {
  url_id: number;
  original_url: string;
  short_code: string;
  user_id: number | null;
  click_count: number;
  created_at: string;
  expires_at: string | null;
  is_active: boolean;
}

interface URLResponse {
  data: URL[];
  meta: {
    total: number;
    page: number;
    limit: number;
    totalPages: number;
    query?: string;
  };
}

const limit = 10;

const getURLs = async (
  page: number,
  query: string,
  errors: { auth: string; admin: string; load: string },
) => {
  const endpoint = query.trim()
    ? `/api/admin/search?q=${encodeURIComponent(query)}&page=${page}&limit=${limit}`
    : `/api/admin/urls?page=${page}&limit=${limit}`;
  const response = await fetch(apiUrl(endpoint), { credentials: "include" });

  if (!response.ok) {
    if (response.status === 401) throw new Error(errors.auth);
    if (response.status === 403) throw new Error(errors.admin);
    throw new Error(errors.load);
  }

  return (await response.json()) as URLResponse;
};

export default function URLList() {
  const { t, formatNumber } = useLocale();
  const [urls, setURLs] = useState<URL[]>([]);
  const [page, setPage] = useState(1);
  const [totalURLs, setTotalURLs] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState("");
  const [searchInput, setSearchInput] = useState("");
  const [detailsItem, setDetailsItem] = useState<URL | null>(null);

  const fetchURLs = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const result = await getURLs(page, searchQuery, {
        auth: t("admin.authRequired"),
        admin: t("admin.adminRequired"),
        load: t("admin.loadError"),
      });
      setURLs(result.data);
      setTotalURLs(result.meta.total);
    } catch (requestError) {
      setError(
        requestError instanceof Error
          ? requestError.message
          : t("admin.loadError"),
      );
    } finally {
      setLoading(false);
    }
  }, [page, searchQuery, t]);

  useEffect(() => {
    const timer = window.setTimeout(() => {
      setSearchQuery(searchInput);
      setPage(1);
    }, 350);
    return () => window.clearTimeout(timer);
  }, [searchInput]);

  useEffect(() => {
    void fetchURLs();
  }, [fetchURLs]);

  useEffect(() => {
    if (page > 1 && Math.ceil(totalURLs / limit) < page) setPage(1);
  }, [page, totalURLs]);

  const handleSearchSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setSearchQuery(searchInput);
    setPage(1);
  };

  const handleDelete = async (shortCode: string) => {
    const response = await fetch(apiUrl(`/api/admin/urls/${shortCode}`), {
      method: "DELETE",
      credentials: "include",
    });
    if (!response.ok) {
      setError(t("admin.deleteError"));
      return;
    }
    await fetchURLs();
  };

  const totalPages = Math.max(1, Math.ceil(totalURLs / limit));

  return (
    <section className={styles.url_list} aria-labelledby="url-index-title">
      <div className={styles.url_list_header}>
        <div>
          <span>{t("admin.label")}</span>
          <h1 id="url-index-title">{t("admin.title")}</h1>
          <p>{t("admin.count", { count: formatNumber(totalURLs) })}</p>
        </div>

        <form
          onSubmit={handleSearchSubmit}
          className={styles.search_form}
          role="search"
        >
          <label htmlFor="url-search">{t("admin.search")}</label>
          <div className={styles.search_container}>
            <input
              id="url-search"
              type="search"
              value={searchInput}
              onChange={(event) => setSearchInput(event.target.value)}
              placeholder={t("admin.searchPlaceholder")}
              className={styles.search_input}
            />
            {searchInput && (
              <button
                type="button"
                onClick={() => {
                  setSearchInput("");
                  setSearchQuery("");
                  setPage(1);
                }}
                className={styles.clear_button}
                aria-label={t("admin.clearSearch")}
              >
                ×
              </button>
            )}
          </div>
        </form>
      </div>

      {error && (
        <div className={styles.error} role="alert">
          {error}
        </div>
      )}

      <div className={styles.table_wrap}>
        <div className={styles.table}>
          <div className={styles.option_names} aria-hidden="true">
            <span>{t("admin.number")}</span>
            <span>{t("admin.original")}</span>
            <span>{t("admin.code")}</span>
            <span>{t("admin.user")}</span>
            <span>{t("admin.clicks")}</span>
            <span>{t("admin.created")}</span>
            <span>{t("admin.shortLink")}</span>
            <span />
          </div>

          {loading ? (
            <div className={styles.loading}>{t("admin.loadingIndex")}</div>
          ) : urls.length === 0 ? (
            <div className={styles.no_results}>
              {searchQuery ? t("admin.noMatch") : t("admin.noLinks")}
            </div>
          ) : (
            urls.map((url, index) => (
              <URLItem
                key={url.url_id}
                number={(page - 1) * limit + index + 1}
                url={url}
                onDelete={handleDelete}
                onManage={() => setDetailsItem(url)}
              />
            ))
          )}
        </div>
      </div>

      {!loading && urls.length > 0 && (
        <Pagination
          page={page}
          totalPages={totalPages}
          loading={loading}
          setPage={setPage}
        />
      )}
      <LinkDetailsModal
        item={detailsItem}
        onClose={() => setDetailsItem(null)}
        onUpdated={async () => {
          await fetchURLs();
          setDetailsItem(null);
        }}
      />
    </section>
  );
}
