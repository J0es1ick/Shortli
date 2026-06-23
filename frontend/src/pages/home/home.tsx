import ShortenerForm from "../../components/shortenerForm/shortenerForm";
import Footer from "../../components/UI/footer/footer";
import { Header } from "../../components/UI/header/header";
import UserHistory from "../../components/userHistory/userHistory";
import LoginModal from "../../components/UI/loginModal/loginModal";
import { useUser } from "../../context/UserContext";
import { useState, useEffect } from "react";
import { useLocation } from "react-router-dom";
import styles from "./home.module.css";

export default function Home() {
  const { user, isLoading } = useUser();
  const [isLoginModalOpen, setIsLoginModalOpen] = useState(false);
  const location = useLocation();

  useEffect(() => {
    if (location.state?.openLoginModal) {
      setIsLoginModalOpen(true);
      window.history.replaceState({}, document.title);
    }
  }, [location]);

  if (isLoading) {
    return <div className={styles.loading}>Loading...</div>;
  }

  return (
    <div>
      <Header />
      <main className={styles.main}>
        <ShortenerForm />

        {user ? (
          <UserHistory />
        ) : (
          <div className={styles.history_prompt}>
            <div className={styles.prompt_content}>
              <h2>Save Your Link History</h2>
              <p>
                Register to keep track of all your shortened links and access
                them anytime.
              </p>
              <div className={styles.prompt_actions}>
                <a href="/register" className={styles.signup_button}>
                  Sign Up Now
                </a>
                <span>or</span>
                <a
                  onClick={() => setIsLoginModalOpen(true)}
                  className={styles.signin_link}
                >
                  Sign In
                </a>
              </div>
            </div>
          </div>
        )}
      </main>
      <Footer />
      <LoginModal
        isOpen={isLoginModalOpen}
        onClose={() => setIsLoginModalOpen(false)}
      />
    </div>
  );
}
