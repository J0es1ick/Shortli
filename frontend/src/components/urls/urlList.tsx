import React, { useState, useEffect, useCallback } from "react";
import URLItem from "./urlItem";
import styles from "./urlList.module.css";
import Pagination from "../UI/pagination/pagination";

export interface URL {
  url_id: number;
  original_url: string;
  short_code: string;
  user_id: number | null;
  click_count: number;
  created_at: Date;
}

export interface URLResponse {
  data: URL[];
  meta: {
    total: number;
    page: number;
    limit: number;
    totalPages: number;
    query?: string;
  };
}

const BASE_URL = "http://localhost:8088";

const getURLs = async (
  page: number = 1,
  limit: number = 10,
  query: string = "",
): Promise<{ data: URL[]; total: number }> => {
  try {
    let url: string;

    if (query.trim()) {
      url = `${BASE_URL}/api/admin/search?q=${encodeURIComponent(
        query,
      )}&page=${page}&limit=${limit}`;
    } else {
      url = `${BASE_URL}/api/admin/urls?page=${page}&limit=${limit}`;
    }

    const response = await fetch(url, {
      credentials: "include",
    });

    if (!response.ok) {
      if (response.status === 401) {
        throw new Error("Authentication required");
      }
      if (response.status === 403) {
        throw new Error("Admin access required");
      }
      throw new Error(`HTTP error! status: ${response.status}`);
    }

    const result: URLResponse = await response.json();

    return {
      data: result.data,
      total: result.meta.total,
    };
  } catch (err) {
    console.error("Error fetching URLs:", err);
    throw err;
  }
};

export default function URLList() {
  const [urls, setURLs] = useState<URL[]>([]);
  const [page, setPage] = useState(1);
  const [totalURLs, setTotalURLs] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState("");
  const [searchInput, setSearchInput] = useState("");
  const limit: number = 10;

  const fetchURLs = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const { data, total } = await getURLs(page, limit, searchQuery);
      setURLs(data);
      setTotalURLs(total);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load URLs");
    } finally {
      setLoading(false);
    }
  }, [page, searchQuery]);

  useEffect(() => {
    const timer = setTimeout(() => {
      setSearchQuery(searchInput);
      setPage(1);
    }, 500);

    return () => clearTimeout(timer);
  }, [searchInput]);

  useEffect(() => {
    let isMounted = true;

    const loadURLs = async () => {
      await fetchURLs();
    };

    if (isMounted) {
      loadURLs();
    }

    return () => {
      isMounted = false;
    };
  }, [fetchURLs]);

  useEffect(() => {
    if (page > 1 && Math.ceil(totalURLs / limit) < page) {
      setPage(1);
    }
  }, [totalURLs, page]);

  const handleSearchSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    setSearchQuery(searchInput);
    setPage(1);
  };

  const handleClearSearch = () => {
    setSearchInput("");
    setSearchQuery("");
    setPage(1);
  };

  return (
    <div className={styles.url_list}>
      <div className={styles.url_list_header}>
        <h1>URL Management</h1>

        <form onSubmit={handleSearchSubmit} className={styles.search_form}>
          <div className={styles.search_container}>
            <input
              type="text"
              value={searchInput}
              onChange={(e) => setSearchInput(e.target.value)}
              placeholder="Search by URL or short code..."
              className={styles.search_input}
            />
            {searchInput && (
              <button
                type="button"
                onClick={handleClearSearch}
                className={styles.clear_button}
                title="Clear search"
              >
                ×
              </button>
            )}
          </div>
        </form>
      </div>

      {error && <div className={styles.error}>{error}</div>}

      <div className={styles.table}>
        <div className={styles.option_names}>
          <button>Number</button>
          <button>Original URL</button>
          <button>Short Code</button>
          <button>User ID</button>
          <button>Click count</button>
          <button>Created at</button>
          <button>Link</button>
        </div>

        {loading ? (
          <div className={styles.loading}>Loading URLs...</div>
        ) : urls.length === 0 ? (
          <div className={styles.no_results}>
            {searchQuery
              ? "No URLs found matching your search."
              : "No URLs found."}
          </div>
        ) : (
          urls.map((url, index) => (
            <URLItem
              key={url.url_id}
              number={page === 1 ? index + 1 : index + 1 + limit * (page - 1)}
              url={url}
            />
          ))
        )}
      </div>

      {!loading && urls.length > 0 && (
        <Pagination
          page={page}
          totalPages={Math.ceil(totalURLs / limit)}
          loading={loading}
          setPage={setPage}
        />
      )}
    </div>
  );
}
