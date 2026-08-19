import React from "react";
import styles from "./pagination.module.css";
import { useLocale } from "../../../context/LocaleContext";

interface IProps {
  page: number;
  totalPages: number;
  loading: boolean;
  setPage: React.Dispatch<React.SetStateAction<number>>;
}

export default function Pagination({
  page,
  loading,
  totalPages,
  setPage,
}: IProps) {
  const { t } = useLocale();
  const handlePageChange = (newPage: number) => {
    if (newPage >= 1 && newPage <= totalPages) {
      setPage(newPage);
    }
  };

  return (
    <div className={styles.pagination}>
      <button
        type="button"
        disabled={page <= 1 || loading}
        onClick={() => handlePageChange(1)}
        aria-label={t("pagination.first")}
      >
        {"<<"}
      </button>
      <button
        type="button"
        disabled={page <= 1 || loading}
        onClick={() => handlePageChange(page - 1)}
        aria-label={t("pagination.previous")}
      >
        {"<"}
      </button>
      <span>{t("pagination.pageOf", { page, total: totalPages })}</span>
      <button
        type="button"
        disabled={page >= totalPages || loading}
        onClick={() => handlePageChange(page + 1)}
        aria-label={t("pagination.next")}
      >
        {">"}
      </button>
      <button
        type="button"
        disabled={page >= totalPages || loading}
        onClick={() => handlePageChange(totalPages)}
        aria-label={t("pagination.last")}
      >
        {">>"}
      </button>
    </div>
  );
}
