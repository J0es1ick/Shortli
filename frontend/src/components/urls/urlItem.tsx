import type { URL } from "./urlList";
import styles from "./urlItem.module.css";

interface IProps {
  number: number;
  url: URL;
}

export default function URLItem({ number, url }: IProps) {
  return (
    <div className={styles.url_item}>
      <h1>{number}</h1>
      <h1>{url.original_url}</h1>
      <h1>{url.short_code}</h1>
      <h1>{String(url.user_id) === "null" ? "Guest" : url.user_id}</h1>
      <h1>{String(url.click_count) === "undefined" ? "0" : url.click_count}</h1>
      <h1>{new Date(url.created_at).toLocaleString()}</h1>
      <a
        href={`http://localhost:8088/${url.short_code.split("/").pop()}`}
        target="_blank"
        rel="noopener noreferrer"
      >
        {`${window.location.host}/${url.short_code.split("/").pop()}`}
      </a>
      <button>
        <img src="public/delete.svg" alt="delete" />
      </button>
    </div>
  );
}
