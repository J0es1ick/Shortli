import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { useUser } from "../../context/UserContext";
import styles from "./signUp.module.css";

export default function SignUpPage() {
  const navigate = useNavigate();
  const { login } = useUser();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const handleRegister = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError("");

    if (password !== confirmPassword) {
      setError("Passwords do not match");
      setLoading(false);
      return;
    }

    if (password.length < 6) {
      setError("Password must be at least 6 characters long");
      setLoading(false);
      return;
    }

    try {
      const registerResponse = await fetch(
        "http://localhost:8088/api/register",
        {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
          },
          credentials: "include",
          body: JSON.stringify({
            email: email,
            password: password,
          }),
        }
      );

      if (!registerResponse.ok) {
        const errorData = await registerResponse.json();
        throw new Error(errorData.error || "Registration failed");
      }

      await registerResponse.json();

      const loginResponse = await fetch("http://localhost:8088/api/login", {
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

      if (!loginResponse.ok) {
        throw new Error("Auto-login after registration failed");
      }

      const loginData = await loginResponse.json();
      login(loginData);

      navigate("/");
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : "An error occurred during registration"
      );
    } finally {
      setLoading(false);
    }
  };

  const handleBackClick = () => {
    navigate("/");
  };

  const handleSignInClick = () => {
    navigate("/", { state: { openLoginModal: true } });
  };

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === "Enter") {
      handleRegister(e);
    }
  };

  return (
    <div className={styles.signup_page}>
      <button className={styles.back_button} onClick={handleBackClick}>
        <svg
          width="20"
          height="20"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
        >
          <path d="M19 12H5M12 19l-7-7 7-7" />
        </svg>
        Back
      </button>

      <div className={styles.form_container}>
        <div className={styles.form_wrapper}>
          <h1 className={styles.page_title}>Create your account</h1>

          <form className={styles.form} onSubmit={handleRegister}>
            <div className={styles.form_group}>
              <label className={styles.form_label}>Email</label>
              <input
                type="email"
                className={styles.form_input}
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                onKeyPress={handleKeyPress}
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
                onKeyPress={handleKeyPress}
                placeholder="Create a password"
                required
                disabled={loading}
                minLength={6}
              />
              <div className={styles.password_hint}>
                Password must be at least 6 characters long
              </div>
            </div>

            <div className={styles.form_group}>
              <label className={styles.form_label}>Confirm Password</label>
              <input
                type="password"
                className={styles.form_input}
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                onKeyPress={handleKeyPress}
                placeholder="Confirm your password"
                required
                disabled={loading}
              />
            </div>

            <button
              type="submit"
              className={styles.submit_button}
              disabled={loading}
            >
              {loading ? "Creating account..." : "Sign up"}
            </button>

            {error && <div className={styles.error_message}>{error}</div>}

            <div className={styles.login_link}>
              <span>Already have an account? </span>
              <a
                type="button"
                onClick={handleSignInClick}
                className={styles.link}
              >
                Sign in
              </a>
            </div>
          </form>
        </div>
      </div>
    </div>
  );
}
