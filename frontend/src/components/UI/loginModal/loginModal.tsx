import { useState, useRef, useEffect } from "react";
import { useUser } from "../../../context/UserContext.tsx";
import styles from "./loginModal.module.css";

interface LoginResponse {
  user_id: number;
  email: string;
  is_admin: boolean;
  created_at: string;
}

interface LoginModalProps {
  isOpen: boolean;
  onClose: () => void;
  onLoginSuccess?: () => void;
}

export default function LoginModal({
  isOpen,
  onClose,
  onLoginSuccess,
}: LoginModalProps) {
  const { login } = useUser();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");

  const modalRef = useRef<HTMLDivElement>(null);
  const buttonRef = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (
        modalRef.current &&
        !modalRef.current.contains(event.target as Node) &&
        buttonRef.current &&
        !buttonRef.current.contains(event.target as Node)
      ) {
        onClose();
        setError("");
        setSuccess("");
      }
    }

    if (isOpen) {
      document.addEventListener("mousedown", handleClickOutside);
    }

    return () => {
      document.removeEventListener("mousedown", handleClickOutside);
    };
  }, [isOpen, onClose]);

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError("");
    setSuccess("");

    try {
      const response = await fetch("http://localhost:8088/api/login", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
        body: JSON.stringify({
          email: email,
          password: password,
        }),
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || "Login failed");
      }

      const data: LoginResponse = await response.json();
      login(data);
      setSuccess(`Welcome back, ${data.email}!`);

      setTimeout(() => {
        onClose();
        setSuccess("");
        setEmail("");
        setPassword("");
        if (onLoginSuccess) onLoginSuccess();
        window.location.reload();
      }, 1000);
    } catch (err) {
      setError(err instanceof Error ? err.message : "An error occurred");
    } finally {
      setLoading(false);
    }
  };

  if (!isOpen) return null;

  return (
    <>
      <div className={styles.modal_overlay}>
        <div ref={modalRef} className={styles.signin_modal}>
          <div className={styles.modal_header}>
            <h3 className={styles.modal_title}>Sign in</h3>
            <button
              onClick={() => {
                onClose();
                setError("");
                setSuccess("");
              }}
              className={styles.close_button}
            >
              ×
            </button>
          </div>
          <form onSubmit={handleLogin}>
            <div className={styles.form_group}>
              <label className={styles.form_label}>Email</label>
              <input
                type="email"
                className={styles.form_input}
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder="Enter your email"
                required
                disabled={loading}
              />
            </div>
            <div className={styles.form_group}>
              <label className={styles.form_label}>Password</label>
              <input
                type="password"
                className={styles.form_input}
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="Enter your password"
                required
                disabled={loading}
              />
            </div>
            <button
              type="submit"
              className={styles.submit_button}
              disabled={loading}
            >
              {loading ? "Signing in..." : "Sign in"}
            </button>

            {error && <div className={styles.error_message}>{error}</div>}
            {success && <div className={styles.success_message}>{success}</div>}

            <div className={styles.register}>
              <span>
                Don't have an account? <a href="/register">Sign up</a>
              </span>
            </div>
          </form>
        </div>
      </div>
    </>
  );
}
