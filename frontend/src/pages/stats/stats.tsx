import { useNavigate } from "react-router-dom";
import Footer from "../../components/UI/footer/footer";
import { Header } from "../../components/UI/header/header";
import URLList from "../../components/urls/urlList";
import styles from "./stats.module.css";
import { useUser } from "../../context/UserContext";
import { useEffect } from "react";

export default function Stats() {
  const { user, isLoading } = useUser();
  const navigate = useNavigate();

  useEffect(() => {
    if (!isLoading && (!user || !user.is_admin)) {
      navigate("/");
    }
  }, [user, isLoading, navigate]);

  if (isLoading) {
    return <div>Loading...</div>;
  }

  if (!user || !user.is_admin) {
    return <div>Access denied</div>;
  }

  return (
    <div>
      <Header />
      <main className={styles.main}>
        <URLList />
      </main>
      <Footer />
    </div>
  );
}
