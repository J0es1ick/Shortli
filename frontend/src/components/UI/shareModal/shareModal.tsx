import { useState } from "react";
import styles from "./shareModal.module.css";

interface ShareModalProps {
  isOpen: boolean;
  onClose: () => void;
  url: string;
  title?: string;
  description?: string;
}

export default function ShareModal({
  isOpen,
  onClose,
  url,
  title = "Check out this short link!",
  description = "Shortened URL created with Shortli",
}: ShareModalProps) {
  const [copied, setCopied] = useState(false);

  if (!isOpen) return null;

  const encodedUrl = encodeURIComponent(url);
  const encodedTitle = encodeURIComponent(title);
  const encodedDescription = encodeURIComponent(description);

  const sharePlatforms = [
    {
      name: "Twitter",
      icon: "🐦",
      color: "#1DA1F2",
      url: `https://twitter.com/intent/tweet?text=${encodedTitle}&url=${encodedUrl}`,
    },
    {
      name: "Facebook",
      icon: "📘",
      color: "#1877F2",
      url: `https://www.facebook.com/sharer/sharer.php?u=${encodedUrl}`,
    },
    {
      name: "LinkedIn",
      icon: "💼",
      color: "#0077B5",
      url: `https://www.linkedin.com/shareArticle?mini=true&url=${encodedUrl}&title=${encodedTitle}&summary=${encodedDescription}`,
    },
    {
      name: "Telegram",
      icon: "📲",
      color: "#0088CC",
      url: `https://t.me/share/url?url=${encodedUrl}&text=${encodedTitle}`,
    },
    {
      name: "WhatsApp",
      icon: "💚",
      color: "#25D366",
      url: `https://wa.me/?text=${encodedTitle}%20${encodedUrl}`,
    },
    {
      name: "Reddit",
      icon: "👽",
      color: "#FF5700",
      url: `https://reddit.com/submit?url=${encodedUrl}&title=${encodedTitle}`,
    },
    {
      name: "Copy Link",
      icon: "🔗",
      color: "#6B7280",
      action: "copy",
    },
  ];

  const handleCopy = () => {
    navigator.clipboard.writeText(url);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const handleShare = async (platform: (typeof sharePlatforms)[0]) => {
    if (platform.action === "copy") {
      handleCopy();
      return;
    }

    const width = 600;
    const height = 400;
    const left = (window.innerWidth - width) / 2;
    const top = (window.innerHeight - height) / 2;

    window.open(
      platform.url,
      "share",
      `width=${width},height=${height},left=${left},top=${top}`,
    );
  };

  const handleNativeShare = async () => {
    // Используем более безопасную проверку
    if (navigator.share && typeof navigator.share === "function") {
      try {
        await navigator.share({
          title,
          text: description,
          url,
        });
      } catch (error) {
        // Пользователь отменил шаринг или произошла ошибка
        if ((error as Error).name !== "AbortError") {
          console.error("Error sharing:", error);
        }
      }
    }
  };

  // Проверяем, поддерживает ли браузер Web Share API
  const supportsNativeShare =
    navigator.share && typeof navigator.share === "function";

  return (
    <div className={styles.modal_overlay}>
      <div className={styles.modal}>
        <div className={styles.modal_header}>
          <h2>Share This Link</h2>
          <button onClick={onClose} className={styles.close_button}>
            ×
          </button>
        </div>

        <div className={styles.url_preview}>
          <div className={styles.url_text}>{url}</div>
          <button
            onClick={handleCopy}
            className={`${styles.copy_button} ${copied ? styles.copied : ""}`}
          >
            {copied ? "✓ Copied" : "Copy"}
          </button>
        </div>

        {supportsNativeShare && (
          <div className={styles.native_share}>
            <button
              onClick={handleNativeShare}
              className={styles.native_share_button}
            >
              <svg
                className={styles.share_icon}
                fill="none"
                viewBox="0 0 24 24"
              >
                <path
                  stroke="currentColor"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth="2"
                  d="M8.684 13.342C8.886 12.938 9 12.482 9 12c0-.482-.114-.938-.316-1.342m0 2.684a3 3 0 110-2.684m0 2.684l6.632 3.316m-6.632-6l6.632-3.316m0 0a3 3 0 105.367-2.684 3 3 0 00-5.367 2.684zm0 9.316a3 3 0 105.368 2.684 3 3 0 00-5.368-2.684z"
                />
              </svg>
              Share via Device
            </button>
          </div>
        )}

        <div className={styles.platforms}>
          <h3>Share on Social Media</h3>
          <div className={styles.platforms_grid}>
            {sharePlatforms.map((platform) => (
              <button
                key={platform.name}
                className={styles.platform_button}
                onClick={() => handleShare(platform)}
                style={
                  { "--platform-color": platform.color } as React.CSSProperties
                }
              >
                <div
                  className={styles.platform_icon}
                  style={{ backgroundColor: platform.color }}
                >
                  {platform.icon}
                </div>
                <span className={styles.platform_name}>{platform.name}</span>
              </button>
            ))}
          </div>
        </div>

        <div className={styles.embed_section}>
          <h3>Embed Options</h3>
          <div className={styles.embed_options}>
            <div className={styles.embed_option}>
              <h4>HTML Link</h4>
              <code className={styles.embed_code}>
                {`<a href="${url}" target="_blank">${title}</a>`}
              </code>
              <button
                onClick={() =>
                  navigator.clipboard.writeText(
                    `<a href="${url}" target="_blank">${title}</a>`,
                  )
                }
                className={styles.copy_code_button}
              >
                Copy HTML
              </button>
            </div>
            <div className={styles.embed_option}>
              <h4>Markdown</h4>
              <code className={styles.embed_code}>{`[${title}](${url})`}</code>
              <button
                onClick={() =>
                  navigator.clipboard.writeText(`[${title}](${url})`)
                }
                className={styles.copy_code_button}
              >
                Copy Markdown
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
