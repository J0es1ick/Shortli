import { useState } from "react";
import styles from "./shortenerForm.module.css";
import ShareModal from "../UI/shareModal/shareModal";
import QRCustomizerModal from "../UI/qrCustomizerModal/qrCustomizerModal";

interface ShortenResponse {
  original_url: string;
  short_code: string;
  short_url: string;
  qr_code_base64: string;
}

export default function ShortenerForm() {
  const [url, setUrl] = useState("");
  const [result, setResult] = useState<ShortenResponse | null>(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const [qrModalOpen, setQrModalOpen] = useState(false);
  const [shareModalOpen, setShareModalOpen] = useState(false);
  const [customQR, setCustomQR] = useState<string>("");

  const handleSubmit = async () => {
    if (!url.trim()) {
      setError("Please enter a URL");
      return;
    }

    setLoading(true);
    setError("");

    try {
      const response = await fetch(`/api/shorten`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ original_url: url }),
        credentials: "include",
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(
          errorData.error || `HTTP error! status: ${response.status}`,
        );
      }

      const data = await response.json();
      setResult(data);
      setCustomQR(data.qr_code_base64);
      setError("");
      setUrl("");
    } catch (err) {
      setError(err instanceof Error ? err.message : "An error occurred");
      setResult(null);
    } finally {
      setLoading(false);
    }
  };

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === "Enter") {
      handleSubmit();
    }
  };

  const handleShare = () => {
    if (!result) return;
    setShareModalOpen(true);
  };

  const handleQRClick = () => {
    if (!result) return;
    setQrModalOpen(true);
  };

  const fullShortUrl = result
    ? `http://localhost:8088/${result.short_url.split("/").pop()}`
    : "";

  return (
    <div className={styles.form}>
      <h1>Shorten the link</h1>
      <input
        type="text"
        value={url}
        onChange={(e) => setUrl(e.target.value)}
        onKeyPress={handleKeyPress}
        placeholder="Enter URL"
        disabled={loading}
      />
      <button onClick={handleSubmit} className={styles.form_button}>
        Shorten
      </button>
      {result && (
        <div className={styles.short_url}>
          <div className={styles.short_url_left_side}>
            <div className={styles.short_link}>
              <a
                className={styles.slink}
                href={`http://localhost:8088/${result.short_url
                  .split("/")
                  .pop()}`}
                target="_blank"
                rel="noopener noreferrer"
              >
                {`shortli/${result.short_url.split("/").pop()}`}
              </a>
            </div>
            <div className={styles.buttons}>
              <button
                onClick={() =>
                  navigator.clipboard.writeText(
                    `http://localhost:8088/${result.short_url.split("/").pop()}`,
                  )
                }
              >
                Copy <text>ctrl+c</text>
              </button>
              <button onClick={handleShare} className={styles.action_button}>
                Share
              </button>
              <a
                href={result.qr_code_base64}
                download={`qr-${result.short_code}.png`}
              >
                Download QR-code
              </a>
            </div>
          </div>
          <div className={styles.short_url_right_side}>
            <div className={styles.qr_code} onClick={handleQRClick}>
              <img src={customQR || result.qr_code_base64} alt="QR Code" />
            </div>
          </div>
        </div>
      )}
      {error && <p style={{ color: "red" }}>{error}</p>}
      <QRCustomizerModal
        isOpen={qrModalOpen}
        onClose={() => setQrModalOpen(false)}
        qrCode={
          result
            ? {
                base64: result.qr_code_base64,
                shortCode: result.short_code,
              }
            : null
        }
        onUpdateQRCode={setCustomQR}
      />

      <ShareModal
        isOpen={shareModalOpen}
        onClose={() => setShareModalOpen(false)}
        url={fullShortUrl}
        title="Check out this short link!"
        description={`Shortened URL created with Shortli: ${fullShortUrl}`}
      />
    </div>
  );
}
