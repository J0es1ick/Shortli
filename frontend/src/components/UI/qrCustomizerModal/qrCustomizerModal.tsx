import { useEffect, useId, useRef, useState } from "react";
import { createPortal } from "react-dom";
import styles from "./qrCustomizerModal.module.css";
import { useLocale } from "../../../context/LocaleContext";
import type { TranslationKey } from "../../../i18n/translations";

interface QRCode {
  base64: string;
  shortCode: string;
}

interface QRCustomizerModalProps {
  isOpen: boolean;
  onClose: () => void;
  qrCode: QRCode | null;
  onUpdateQRCode?: (customizedQR: string) => void;
}

const presets = [
  { nameKey: "qr.preset.ink", foreground: "#111111", background: "#ffffff" },
  {
    nameKey: "qr.preset.reverse",
    foreground: "#f3f3ee",
    background: "#111111",
  },
  { nameKey: "qr.preset.signal", foreground: "#0c5b46", background: "#e7f4e8" },
  { nameKey: "qr.preset.cobalt", foreground: "#163c8c", background: "#edf1ff" },
  { nameKey: "qr.preset.ember", foreground: "#8f2f20", background: "#fff0e8" },
  { nameKey: "qr.preset.plum", foreground: "#5e255e", background: "#f7eafa" },
] satisfies Array<{
  nameKey: TranslationKey;
  foreground: string;
  background: string;
}>;

const loadImage = (source: string, errorMessage: string) =>
  new Promise<HTMLImageElement>((resolve, reject) => {
    const image = new Image();
    image.onload = () => resolve(image);
    image.onerror = () => reject(new Error(errorMessage));
    image.src = source;
  });

const hexToRgb = (hex: string) => {
  const normalized = hex.replace("#", "");
  if (!/^[0-9a-f]{6}$/i.test(normalized)) return null;
  return {
    r: Number.parseInt(normalized.slice(0, 2), 16),
    g: Number.parseInt(normalized.slice(2, 4), 16),
    b: Number.parseInt(normalized.slice(4, 6), 16),
  };
};

export default function QRCustomizerModal({
  isOpen,
  onClose,
  qrCode,
  onUpdateQRCode,
}: QRCustomizerModalProps) {
  const { t } = useLocale();
  const [foregroundColor, setForegroundColor] = useState("#111111");
  const [backgroundColor, setBackgroundColor] = useState("#ffffff");
  const [logo, setLogo] = useState<string | null>(null);
  const [customizedQR, setCustomizedQR] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [copied, setCopied] = useState(false);
  const uploadId = useId();
  const previewCanvasRef = useRef<HTMLCanvasElement>(null);
  const baseQR = qrCode?.base64;

  useEffect(() => {
    if (!isOpen) return;

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") onClose();
    };
    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    document.addEventListener("keydown", handleKeyDown);

    return () => {
      document.body.style.overflow = previousOverflow;
      document.removeEventListener("keydown", handleKeyDown);
    };
  }, [isOpen, onClose]);

  useEffect(() => {
    if (!isOpen || !baseQR) return;

    let cancelled = false;
    const renderQR = async () => {
      const foreground = hexToRgb(foregroundColor);
      const background = hexToRgb(backgroundColor);
      if (!foreground || !background) return;

      setLoading(true);
      setError("");

      try {
        const sourceImage = await loadImage(baseQR, t("qr.imageLoadError"));
        const size = Math.max(sourceImage.naturalWidth, 512);
        const canvas = previewCanvasRef.current;
        if (!canvas || cancelled) return;
        canvas.width = size;
        canvas.height = size;
        const context = canvas.getContext("2d", { willReadFrequently: true });
        if (!context) throw new Error(t("qr.previewUnavailable"));

        context.imageSmoothingEnabled = false;
        context.drawImage(sourceImage, 0, 0, size, size);
        const imageData = context.getImageData(0, 0, size, size);

        for (let index = 0; index < imageData.data.length; index += 4) {
          const average =
            (imageData.data[index] +
              imageData.data[index + 1] +
              imageData.data[index + 2]) /
            3;
          const color = average < 145 ? foreground : background;
          imageData.data[index] = color.r;
          imageData.data[index + 1] = color.g;
          imageData.data[index + 2] = color.b;
          imageData.data[index + 3] = 255;
        }
        context.putImageData(imageData, 0, 0);

        if (logo) {
          const logoImage = await loadImage(logo, t("qr.imageLoadError"));
          const logoBoxSize = Math.round(size * 0.14);
          const padding = Math.round(size * 0.02);
          const safeZoneSize = logoBoxSize + padding * 2;
          const safeZoneX = Math.round((size - safeZoneSize) / 2);
          const safeZoneY = Math.round((size - safeZoneSize) / 2);
          const logoScale = Math.min(
            logoBoxSize / logoImage.naturalWidth,
            logoBoxSize / logoImage.naturalHeight,
          );
          const logoWidth = Math.max(
            1,
            Math.round(logoImage.naturalWidth * logoScale),
          );
          const logoHeight = Math.max(
            1,
            Math.round(logoImage.naturalHeight * logoScale),
          );
          const logoX = Math.round((size - logoWidth) / 2);
          const logoY = Math.round((size - logoHeight) / 2);

          context.fillStyle = backgroundColor;
          context.fillRect(safeZoneX, safeZoneY, safeZoneSize, safeZoneSize);
          context.imageSmoothingEnabled = true;
          context.drawImage(logoImage, logoX, logoY, logoWidth, logoHeight);
        }

        const nextQR = canvas.toDataURL("image/png");
        if (!cancelled) {
          setCustomizedQR(nextQR);
          onUpdateQRCode?.(nextQR);
        }
      } catch (generationError) {
        if (!cancelled) {
          setCustomizedQR(baseQR);
          setError(
            generationError instanceof Error
              ? generationError.message
              : t("qr.previewError"),
          );
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    };

    void renderQR();

    return () => {
      cancelled = true;
    };
  }, [
    backgroundColor,
    baseQR,
    foregroundColor,
    isOpen,
    logo,
    onUpdateQRCode,
    t,
  ]);

  const handleLogoUpload = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) return;
    if (!file.type.startsWith("image/") || file.size > 2 * 1024 * 1024) {
      setError(t("qr.fileError"));
      event.target.value = "";
      return;
    }

    const reader = new FileReader();
    reader.onload = () => setLogo(String(reader.result));
    reader.onerror = () => setError(t("qr.readError"));
    reader.readAsDataURL(file);
  };

  const handleDownload = () => {
    if (!customizedQR || !qrCode) return;
    const link = document.createElement("a");
    link.href = customizedQR;
    link.download = `shortli-${qrCode.shortCode}.png`;
    link.click();
  };

  const handleCopy = async () => {
    if (!customizedQR) return;
    try {
      const blob = await (await fetch(customizedQR)).blob();
      await navigator.clipboard.write([
        new ClipboardItem({ [blob.type || "image/png"]: blob }),
      ]);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1600);
    } catch {
      setError(t("qr.copyUnsupported"));
    }
  };

  if (!isOpen || !qrCode) return null;

  return createPortal(
    <div
      className={styles.modal_overlay}
      onMouseDown={(event) => {
        if (event.target === event.currentTarget) onClose();
      }}
    >
      <div
        className={styles.modal}
        role="dialog"
        aria-modal="true"
        aria-labelledby="qr-modal-title"
      >
        <div className={styles.modal_header}>
          <div>
            <span>{t("qr.studio")}</span>
            <h2 id="qr-modal-title">{t("qr.title")}</h2>
          </div>
          <button
            onClick={onClose}
            className={styles.close_button}
            aria-label={t("qr.close")}
          >
            ×
          </button>
        </div>

        <div className={styles.modal_content}>
          <section
            className={styles.preview_section}
            aria-label={t("qr.preview")}
          >
            <div className={styles.preview_label}>
              <span>{t("qr.preview")}</span>
              <span aria-busy={loading}>{t("qr.live")}</span>
            </div>
            <div className={styles.qr_preview}>
              <canvas
                ref={previewCanvasRef}
                role="img"
                aria-label={t("qr.alt")}
              />
            </div>
            <p>{t("qr.liveNote")}</p>
            <div className={styles.preview_actions}>
              <button
                onClick={handleDownload}
                className={styles.primary_button}
              >
                {t("qr.download")}
              </button>
              <button onClick={handleCopy} className={styles.secondary_button}>
                {copied ? t("common.copied") : t("qr.copyImage")}
              </button>
            </div>
          </section>

          <section className={styles.controls} aria-label={t("qr.controls")}>
            <div className={styles.control_group}>
              <div className={styles.control_heading}>
                <span>01</span>
                <h3>{t("qr.colorSystem")}</h3>
              </div>
              <div className={styles.color_grid}>
                <label>
                  <span>{t("qr.foreground")}</span>
                  <div className={styles.color_field}>
                    <input
                      type="color"
                      value={foregroundColor}
                      onInput={(event) =>
                        setForegroundColor(event.currentTarget.value)
                      }
                      onChange={(event) =>
                        setForegroundColor(event.currentTarget.value)
                      }
                      aria-label={t("qr.foregroundAria")}
                    />
                    <input
                      type="text"
                      value={foregroundColor}
                      onChange={(event) =>
                        setForegroundColor(event.target.value)
                      }
                      aria-label={t("qr.foregroundHex")}
                      maxLength={7}
                    />
                  </div>
                </label>
                <label>
                  <span>{t("qr.background")}</span>
                  <div className={styles.color_field}>
                    <input
                      type="color"
                      value={backgroundColor}
                      onInput={(event) =>
                        setBackgroundColor(event.currentTarget.value)
                      }
                      onChange={(event) =>
                        setBackgroundColor(event.currentTarget.value)
                      }
                      aria-label={t("qr.backgroundAria")}
                    />
                    <input
                      type="text"
                      value={backgroundColor}
                      onChange={(event) =>
                        setBackgroundColor(event.target.value)
                      }
                      aria-label={t("qr.backgroundHex")}
                      maxLength={7}
                    />
                  </div>
                </label>
              </div>
              <div className={styles.presets}>
                {presets.map((preset) => (
                  <button
                    type="button"
                    key={preset.nameKey}
                    onClick={() => {
                      setForegroundColor(preset.foreground);
                      setBackgroundColor(preset.background);
                    }}
                    aria-label={t("qr.usePreset", { name: t(preset.nameKey) })}
                  >
                    <i
                      style={{
                        background: `linear-gradient(135deg, ${preset.background} 50%, ${preset.foreground} 50%)`,
                      }}
                    />
                    <span>{t(preset.nameKey)}</span>
                  </button>
                ))}
              </div>
            </div>

            <div className={styles.control_group}>
              <div className={styles.control_heading}>
                <span>02</span>
                <h3>{t("qr.centerMark")}</h3>
              </div>
              {logo ? (
                <div className={styles.logo_selected}>
                  <img src={logo} alt={t("qr.uploadedAlt")} />
                  <div>
                    <strong>{t("qr.logoAdded")}</strong>
                    <button type="button" onClick={() => setLogo(null)}>
                      {t("qr.remove")}
                    </button>
                  </div>
                </div>
              ) : (
                <label htmlFor={uploadId} className={styles.upload_label}>
                  <input
                    id={uploadId}
                    type="file"
                    accept="image/png,image/jpeg"
                    onChange={handleLogoUpload}
                  />
                  <span aria-hidden="true">＋</span>
                  <strong>{t("qr.addLogo")}</strong>
                  <small>{t("qr.fileHint")}</small>
                </label>
              )}
            </div>

            {error && (
              <p className={styles.error} role="alert">
                {error}
              </p>
            )}
          </section>
        </div>
      </div>
    </div>,
    document.body,
  );
}
