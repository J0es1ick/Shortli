import React from "react";
import styles from "./pagination.module.css";

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
  const handlePageChange = (newPage: number) => {
    if (newPage >= 1 && newPage <= totalPages) {
      setPage(newPage);
    }
  };

  return (
    <div className={styles.pagination}>
      <button
        disabled={page <= 1 || loading}
        onClick={() => handlePageChange(1)}
      >
        {"<<"}
      </button>
      <button
        disabled={page <= 1 || loading}
        onClick={() => handlePageChange(page - 1)}
      >
        {"<"}
      </button>
      <span>
        {page} of {totalPages}
      </span>
      <button
        disabled={page >= totalPages || loading}
        onClick={() => handlePageChange(page + 1)}
      >
        {">"}
      </button>
      <button
        disabled={page >= totalPages || loading}
        onClick={() => handlePageChange(totalPages)}
      >
        {">>"}
      </button>
    </div>
  );
}
