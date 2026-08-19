import { useCallback, useEffect, useMemo, useState } from "react";
import { useUser } from "../../context/UserContext";
import {
  getUserRole,
  type User,
  type UserRole,
} from "../../lib/userAccess";
import { useLocale } from "../../context/LocaleContext";
import { apiUrl } from "../../lib/urls";
import styles from "./adminUsers.module.css";

interface UsersPayload {
  data: User[];
  meta: {
    total: number;
    page: number;
    limit: number;
    totalPages: number;
    roleCounts: Record<UserRole, number>;
  };
}

const roles: UserRole[] = ["owner", "admin", "support", "user"];

export default function AdminUsers() {
  const { user: currentUser } = useUser();
  const { t, formatDate, formatNumber, apiError } = useLocale();
  const [users, setUsers] = useState<User[]>([]);
  const [drafts, setDrafts] = useState<Record<number, UserRole>>({});
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const [totalPages, setTotalPages] = useState(1);
  const [roleCounts, setRoleCounts] = useState<Record<UserRole, number>>({
    owner: 0,
    admin: 0,
    support: 0,
    user: 0,
  });
  const [query, setQuery] = useState("");
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState<number | null>(null);
  const [saved, setSaved] = useState<number | null>(null);
  const [error, setError] = useState("");

  const load = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const response = await fetch(
        apiUrl(`/api/admin/users?page=${page}&limit=50`),
        { credentials: "include" },
      );
      if (!response.ok) throw new Error(t("admin.usersLoadError"));
      const payload = (await response.json()) as UsersPayload;
      setUsers(payload.data);
      setDrafts(
        Object.fromEntries(
          payload.data.map((entry) => [entry.user_id, getUserRole(entry)]),
        ),
      );
      setTotal(payload.meta.total);
      setTotalPages(Math.max(1, payload.meta.totalPages));
      setRoleCounts(payload.meta.roleCounts);
    } catch (requestError) {
      setError(
        requestError instanceof Error
          ? requestError.message
          : t("admin.usersLoadError"),
      );
    } finally {
      setLoading(false);
    }
  }, [page, t]);

  useEffect(() => {
    void load();
  }, [load]);

  const visibleUsers = useMemo(() => {
    const normalized = query.trim().toLowerCase();
    if (!normalized) return users;
    return users.filter((entry) =>
      entry.email.toLowerCase().includes(normalized),
    );
  }, [query, users]);

  const currentRole = currentUser ? getUserRole(currentUser) : "user";

  const canEdit = (entry: User) =>
    entry.user_id !== currentUser?.user_id &&
    (currentRole === "owner" || getUserRole(entry) !== "owner");

  const availableRoles = (entry: User) =>
    currentRole === "owner" || getUserRole(entry) === "owner"
      ? roles
      : roles.filter((role) => role !== "owner");

  const saveRole = async (entry: User) => {
    const previousRole = getUserRole(entry);
    const nextRole = drafts[entry.user_id];
    if (!nextRole || nextRole === previousRole) return;
    setBusy(entry.user_id);
    setSaved(null);
    setError("");
    try {
      const response = await fetch(apiUrl(`/api/admin/users/${entry.user_id}`), {
        method: "PUT",
        credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ role: nextRole }),
      });
      if (!response.ok) {
        const payload = (await response.json().catch(() => ({}))) as {
          error?: string;
        };
        throw new Error(apiError(payload.error, "admin.userUpdateError"));
      }
      const updated = (await response.json()) as User;
      setUsers((current) =>
        current.map((item) =>
          item.user_id === updated.user_id ? updated : item,
        ),
      );
      setDrafts((current) => ({
        ...current,
        [updated.user_id]: getUserRole(updated),
      }));
      setRoleCounts((current) => ({
        ...current,
        [previousRole]: Math.max(0, current[previousRole] - 1),
        [nextRole]: current[nextRole] + 1,
      }));
      setSaved(updated.user_id);
      window.setTimeout(() => setSaved(null), 1800);
    } catch (requestError) {
      setError(
        requestError instanceof Error
          ? requestError.message
          : t("admin.userUpdateError"),
      );
    } finally {
      setBusy(null);
    }
  };

  return (
    <section className={styles.section} id="team" aria-labelledby="team-title">
      <div className={styles.heading}>
        <div>
          <span>{t("admin.teamLabel")}</span>
          <h2 id="team-title">{t("admin.teamTitle")}</h2>
          <p>{t("admin.teamDescription", { count: formatNumber(total) })}</p>
        </div>
        <label className={styles.search}>
          <span>{t("admin.userSearch")}</span>
          <input
            type="search"
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder={t("admin.userSearchPlaceholder")}
          />
        </label>
      </div>

      <div className={styles.roleSummary}>
        {roles.map((role) => (
          <div key={role}>
            <strong>{formatNumber(roleCounts[role])}</strong>
            <span>{t(`admin.role.${role}`)}</span>
          </div>
        ))}
      </div>

      {error && (
        <p className={styles.error} role="alert">
          {error}
        </p>
      )}

      {loading ? (
        <div className={styles.empty}>{t("admin.usersLoading")}</div>
      ) : visibleUsers.length === 0 ? (
        <div className={styles.empty}>{t("admin.usersEmpty")}</div>
      ) : (
        <div className={styles.list}>
          {visibleUsers.map((entry, index) => {
            const role = getUserRole(entry);
            const draft = drafts[entry.user_id] ?? role;
            const editable = canEdit(entry);
            return (
              <article key={entry.user_id} className={styles.userCard}>
                <span className={styles.index}>
                  {String((page - 1) * 50 + index + 1).padStart(2, "0")}
                </span>
                <div className={styles.identity}>
                  <strong>{entry.email}</strong>
                  <span>
                    {t("admin.joined")} {formatDate(entry.created_at)}
                  </span>
                </div>
                <label className={styles.roleSelect}>
                  <span>{t("admin.accessRole")}</span>
                  <select
                    value={draft}
                    disabled={!editable || busy === entry.user_id}
                    onChange={(event) =>
                      setDrafts((current) => ({
                        ...current,
                        [entry.user_id]: event.target.value as UserRole,
                      }))
                    }
                  >
                    {availableRoles(entry).map((option) => (
                      <option key={option} value={option}>
                        {t(`admin.role.${option}`)}
                      </option>
                    ))}
                  </select>
                </label>
                <button
                  type="button"
                  className={styles.saveButton}
                  disabled={!editable || busy === entry.user_id || draft === role}
                  onClick={() => void saveRole(entry)}
                >
                  {busy === entry.user_id
                    ? t("admin.savingRole")
                    : saved === entry.user_id
                      ? t("admin.roleSaved")
                      : t("admin.saveRole")}
                </button>
              </article>
            );
          })}
        </div>
      )}

      {totalPages > 1 && (
        <nav className={styles.pagination} aria-label={t("admin.userPages")}>
          <button
            type="button"
            disabled={page <= 1}
            onClick={() => setPage((current) => Math.max(1, current - 1))}
          >
            {t("admin.previous")}
          </button>
          <span>
            {page} / {totalPages}
          </span>
          <button
            type="button"
            disabled={page >= totalPages}
            onClick={() =>
              setPage((current) => Math.min(totalPages, current + 1))
            }
          >
            {t("admin.next")}
          </button>
        </nav>
      )}

      <p className={styles.accessNote}>{t("admin.roleNote")}</p>
    </section>
  );
}
