import { useNavigate } from "react-router-dom";
import Footer from "../../components/UI/footer/footer";
import { Header } from "../../components/UI/header/header";
import NetworkMesh from "../../components/networkMesh/networkMesh";
import URLList from "../../components/urls/urlList";
import styles from "./stats.module.css";
import { useUser } from "../../context/UserContext";
import { useEffect } from "react";
import { useLocale } from "../../context/LocaleContext";
import AbuseReports from "../../components/abuseReports/abuseReports";

export default function Stats() {
  const { user, isLoading } = useUser();
  const { t } = useLocale();
  const navigate = useNavigate();

  useEffect(() => {
    if (!isLoading && (!user || !user.is_admin)) {
      navigate("/");
    }
  }, [user, isLoading, navigate]);

  if (isLoading) {
    return <div>{t("admin.loading")}</div>;
  }

  if (!user || !user.is_admin) {
    return <div>{t("admin.accessDenied")}</div>;
  }

  return (
    <div>
      <NetworkMesh />
      <Header />
      <main className={styles.main}>
        <AbuseReports />
        <URLList />
      </main>
      <Footer />
    </div>
  );
}
