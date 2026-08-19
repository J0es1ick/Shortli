import { useNavigate } from "react-router-dom";
import Footer from "../../components/UI/footer/footer";
import { Header } from "../../components/UI/header/header";
import NetworkMesh from "../../components/networkMesh/networkMesh";
import URLList from "../../components/urls/urlList";
import styles from "./stats.module.css";
import { useUser } from "../../context/UserContext";
import { getUserRole, hasStaffAccess } from "../../lib/userAccess";
import { useEffect } from "react";
import { useLocale } from "../../context/LocaleContext";
import AbuseReports from "../../components/abuseReports/abuseReports";
import AdminUsers from "../../components/adminUsers/adminUsers";

export default function Stats() {
  const { user, isLoading } = useUser();
  const { t } = useLocale();
  const navigate = useNavigate();

  useEffect(() => {
    if (!isLoading && !hasStaffAccess(user)) {
      navigate("/");
    }
  }, [user, isLoading, navigate]);

  if (isLoading) {
    return <div>{t("admin.loading")}</div>;
  }

  if (!user || !hasStaffAccess(user)) {
    return <div>{t("admin.accessDenied")}</div>;
  }

  const role = getUserRole(user);
  const canManageUsers = role === "owner" || role === "admin";

  return (
    <div className={styles.page}>
      <NetworkMesh />
      <Header />
      <main className={styles.main}>
        <section className={styles.hero} aria-labelledby="admin-page-title">
          <div>
            <span>{t("admin.dashboardLabel")}</span>
            <h1 id="admin-page-title">{t("admin.dashboardTitle")}</h1>
            <p>{t("admin.dashboardDescription")}</p>
          </div>
          <div className={styles.roleCard}>
            <span>{t("admin.yourAccess")}</span>
            <strong>{t(`admin.role.${role}`)}</strong>
            <p>{t(`admin.roleDescription.${role}`)}</p>
          </div>
        </section>

        <nav className={styles.sectionNav} aria-label={t("admin.sections")}>
          {canManageUsers && <a href="#team">{t("admin.section.team")}</a>}
          <a href="#moderation">{t("admin.section.moderation")}</a>
          <a href="#links">{t("admin.section.links")}</a>
        </nav>

        {canManageUsers && <AdminUsers />}
        <div id="moderation">
          <AbuseReports />
        </div>
        <div id="links">
          <URLList />
        </div>
      </main>
      <Footer />
    </div>
  );
}
