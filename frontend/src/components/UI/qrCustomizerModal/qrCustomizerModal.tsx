import { useState, useEffect } from "react";
import styles from "./qrCustomizerModal.module.css";

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

export default function QRCustomizerModal({
  isOpen,
  onClose,
  qrCode,
  onUpdateQRCode,
}: QRCustomizerModalProps) {
  const [foregroundColor, setForegroundColor] = useState("#000000");
  const [backgroundColor, setBackgroundColor] = useState("#ffffff");
  const [logo, setLogo] = useState<string | null>(null);
  const [customizedQR, setCustomizedQR] = useState<string>("");
  const [loading, setLoading] = useState(false);

  // Инициализация при открытии модалки
  useEffect(() => {
    if (isOpen && qrCode) {
      setCustomizedQR(qrCode.base64);
    }
  }, [isOpen, qrCode]);

  const handleColorChange = async (
    colorType: "foreground" | "background",
    color: string,
  ) => {
    if (colorType === "foreground") {
      setForegroundColor(color);
    } else {
      setBackgroundColor(color);
    }

    await generateCustomQR();
  };

  const handleLogoUpload = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      const reader = new FileReader();
      reader.onload = (event) => {
        const result = event.target?.result as string;
        setLogo(result);
        generateCustomQR(result);
      };
      reader.readAsDataURL(file);
    }
  };

  const handleRemoveLogo = () => {
    setLogo(null);
    generateCustomQR();
  };

  const generateCustomQR = async (customLogo?: string) => {
    if (!qrCode) return;

    setLoading(true);

    try {
      // Здесь будет API запрос для генерации кастомного QR-кода
      // Пока используем статичную генерацию или имитируем
      const response = await fetch("http://localhost:8088/api/customize-qr", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          short_code: qrCode.shortCode,
          foreground_color: foregroundColor,
          background_color: backgroundColor,
          logo: customLogo || logo,
        }),
      });

      if (response.ok) {
        const data = await response.json();
        setCustomizedQR(data.custom_qr_base64);
        if (onUpdateQRCode) {
          onUpdateQRCode(data.custom_qr_base64);
        }
      }
    } catch (error) {
      console.error("Error generating custom QR:", error);
      // В режиме демо показываем сообщение
      setCustomizedQR(qrCode.base64);
    } finally {
      setLoading(false);
    }
  };

  const handleDownload = () => {
    if (!customizedQR) return;

    const link = document.createElement("a");
    link.href = customizedQR;
    link.download = `shortli-qr-${qrCode?.shortCode}.png`;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
  };

  if (!isOpen || !qrCode) return null;

  return (
    <div className={styles.modal_overlay}>
      <div className={styles.modal}>
        <div className={styles.modal_header}>
          <h2>Customize QR Code</h2>
          <button onClick={onClose} className={styles.close_button}>
            ×
          </button>
        </div>

        <div className={styles.modal_content}>
          <div className={styles.preview_section}>
            <div className={styles.qr_preview}>
              {loading ? (
                <div className={styles.loading}>Generating...</div>
              ) : (
                <img src={customizedQR} alt="Custom QR Code" />
              )}
            </div>
            <div className={styles.preview_actions}>
              <button
                onClick={handleDownload}
                className={styles.download_button}
              >
                Download QR Code
              </button>
              <button
                onClick={() => navigator.clipboard.writeText(customizedQR)}
                className={styles.copy_button}
              >
                Copy QR Image
              </button>
            </div>
          </div>

          <div className={styles.customization_section}>
            <div className={styles.color_picker}>
              <h3>Colors</h3>
              <div className={styles.color_group}>
                <label>Foreground Color</label>
                <div className={styles.color_input_group}>
                  <input
                    type="color"
                    value={foregroundColor}
                    onChange={(e) =>
                      handleColorChange("foreground", e.target.value)
                    }
                    className={styles.color_picker_input}
                  />
                  <input
                    type="text"
                    value={foregroundColor}
                    onChange={(e) =>
                      handleColorChange("foreground", e.target.value)
                    }
                    className={styles.color_text_input}
                  />
                </div>
              </div>
              <div className={styles.color_group}>
                <label>Background Color</label>
                <div className={styles.color_input_group}>
                  <input
                    type="color"
                    value={backgroundColor}
                    onChange={(e) =>
                      handleColorChange("background", e.target.value)
                    }
                    className={styles.color_picker_input}
                  />
                  <input
                    type="text"
                    value={backgroundColor}
                    onChange={(e) =>
                      handleColorChange("background", e.target.value)
                    }
                    className={styles.color_text_input}
                  />
                </div>
              </div>
            </div>

            <div className={styles.logo_section}>
              <h3>Logo</h3>
              {logo ? (
                <div className={styles.logo_preview_container}>
                  <div className={styles.logo_preview}>
                    <img src={logo} alt="Logo" />
                  </div>
                  <button
                    onClick={handleRemoveLogo}
                    className={styles.remove_logo_button}
                  >
                    Remove Logo
                  </button>
                </div>
              ) : (
                <div className={styles.upload_area}>
                  <input
                    type="file"
                    id="logo-upload"
                    accept="image/*"
                    onChange={handleLogoUpload}
                    className={styles.file_input}
                  />
                  <label htmlFor="logo-upload" className={styles.upload_label}>
                    <svg
                      className={styles.upload_icon}
                      fill="none"
                      viewBox="0 0 24 24"
                    >
                      <path
                        stroke="currentColor"
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        strokeWidth="2"
                        d="M12 5v14m-7-7h14"
                      />
                    </svg>
                    <span>Upload Logo</span>
                    <span className={styles.upload_hint}>
                      PNG, JPG up to 2MB
                    </span>
                  </label>
                </div>
              )}
            </div>

            <div className={styles.preset_colors}>
              <h3>Color Presets</h3>
              <div className={styles.color_presets}>
                {[
                  { name: "Classic", fg: "#000000", bg: "#ffffff" },
                  { name: "Dark", fg: "#ffffff", bg: "#1a1a1a" },
                  { name: "Ocean", fg: "#0077be", bg: "#e6f7ff" },
                  { name: "Forest", fg: "#2e7d32", bg: "#f1f8e9" },
                  { name: "Sunset", fg: "#ff6b35", bg: "#fff3e0" },
                  { name: "Berry", fg: "#9c27b0", bg: "#f3e5f5" },
                ].map((preset) => (
                  <button
                    key={preset.name}
                    className={styles.color_preset}
                    onClick={() => {
                      setForegroundColor(preset.fg);
                      setBackgroundColor(preset.bg);
                      generateCustomQR();
                    }}
                    style={{
                      background: `linear-gradient(135deg, ${preset.bg} 50%, ${preset.fg} 50%)`,
                    }}
                    title={`${preset.name}: ${preset.fg} on ${preset.bg}`}
                  >
                    <span className={styles.preset_name}>{preset.name}</span>
                  </button>
                ))}
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
