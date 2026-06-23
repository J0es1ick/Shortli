import { useState, useEffect } from "react";
import { useUser } from "../../context/UserContext";
import styles from "./userHistory.module.css";

interface URL {
  url_id: number;
  original_url: string;
  short_code: string;
  click_count: number;
  created_at: string;
}

interface HistoryResponse {
  data: URL[];
  meta: {
    total: number;
    page: number;
    limit: number;
    totalPages: number;
  };
}

export default function UserHistory() {
  const { user } = useUser();
  const [urls, setUrls] = useState<URL[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [page, setPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const limit = 10;

  const fetchUserHistory = async () => {
    if (!user) return;

    setLoading(true);
    setError("");

    try {
      const response = await fetch(
        `http://localhost:8088/api/history?page=${page}&limit=${limit}`,
        {
          credentials: "include",
        }
      );

      if (!response.ok) {
        throw new Error("Failed to fetch history");
      }

      const data: HistoryResponse = await response.json();
      setUrls(data.data);
      setTotalPages(data.meta.totalPages);
    } catch (err) {
      setError(err instanceof Error ? err.message : "An error occurred");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchUserHistory();
  }, [user, page]);

  const calculateTimeUntilDeletion = (createdAt: string): string => {
    const created = new Date(createdAt);
    const deletionDate = new Date(created);
    deletionDate.setMonth(deletionDate.getMonth() + 1);

    const now = new Date();
    const diffTime = deletionDate.getTime() - now.getTime();
    const diffDays = Math.ceil(diffTime / (1000 * 60 * 60 * 24));

    if (diffDays <= 0) {
      return "Expires today";
    } else if (diffDays === 1) {
      return "1 day left";
    } else if (diffDays <= 7) {
      return `${diffDays} days left`;
    } else {
      const weeks = Math.ceil(diffDays / 7);
      return `${weeks} week${weeks > 1 ? "s" : ""} left`;
    }
  };

  const getExpirationClass = (createdAt: string): string => {
    const created = new Date(createdAt);
    const deletionDate = new Date(created);
    deletionDate.setMonth(deletionDate.getMonth() + 1);

    const now = new Date();
    const diffTime = deletionDate.getTime() - now.getTime();
    const diffDays = Math.ceil(diffTime / (1000 * 60 * 60 * 24));

    if (diffDays <= 3) return styles.expiring_soon;
    if (diffDays <= 7) return styles.expiring;
    return styles.normal;
  };

  if (loading) {
    return <div className={styles.loading}>Loading your history...</div>;
  }

  if (error) {
    return (
      <div className={styles.error}>
        <p>{error}</p>
        <button onClick={fetchUserHistory}>Retry</button>
      </div>
    );
  }

  return (
    <div className={styles.user_history}>
      <h2 className={styles.title}>Your Link History</h2>

      {urls.length === 0 ? (
        <div className={styles.empty_state}>
          <p>You haven't created any shortened links yet.</p>
          <p>Start by shortening your first URL above!</p>
        </div>
      ) : (
        <>
          <div className={styles.history_list}>
            {urls.map((url, index) => (
              <div key={url.url_id} className={styles.history_item}>
                <div className={styles.item_header}>
                  <span className={styles.item_number}>
                    {(page - 1) * limit + index + 1}
                  </span>
                  <span
                    className={`${styles.expiration} ${getExpirationClass(
                      url.created_at
                    )}`}
                  >
                    {calculateTimeUntilDeletion(url.created_at)}
                  </span>
                </div>

                <div className={styles.url_info}>
                  <div className={styles.original_url}>
                    <a
                      href={url.original_url}
                      target="_blank"
                      rel="noopener noreferrer"
                      title={url.original_url}
                    >
                      {url.original_url.length > 60
                        ? `${url.original_url.substring(0, 60)}...`
                        : url.original_url}
                    </a>
                  </div>

                  <div className={styles.short_url}>
                    <a
                      href={`http://localhost:8088/${url.short_code}`}
                      target="_blank"
                      rel="noopener noreferrer"
                    >
                      shortli/{url.short_code}
                    </a>
                    <span className={styles.clicks}>
                      {url.click_count} clicks
                    </span>
                  </div>

                  <div className={styles.created_date}>
                    Created: {new Date(url.created_at).toLocaleDateString()}
                  </div>
                </div>
              </div>
            ))}
          </div>

          {totalPages > 1 && (
            <div className={styles.pagination}>
              <button
                onClick={() => setPage((prev) => Math.max(prev - 1, 1))}
                disabled={page === 1}
                className={styles.pagination_button}
              >
                Previous
              </button>

              <span className={styles.page_info}>
                Page {page} of {totalPages}
              </span>

              <button
                onClick={() =>
                  setPage((prev) => Math.min(prev + 1, totalPages))
                }
                disabled={page === totalPages}
                className={styles.pagination_button}
              >
                Next
              </button>
            </div>
          )}
        </>
      )}
    </div>
  );
}
