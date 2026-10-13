import "./App.css";
import Home from "./pages/home/home";
import { BrowserRouter, HashRouter, Route, Routes } from "react-router-dom";
import { lazy, Suspense, useState } from "react";
import { isDemo } from "./demo/config";
import ScrollToAnchor from "./components/ScrollToAnchor";
import Stats from "./pages/stats/stats";
import SignUpPage from "./pages/signUp/signUp";
import PrivacyPage from "./pages/info/privacy";
import TermsPage from "./pages/info/terms";
import StatusPage from "./pages/info/status";
import DevelopersPage from "./pages/developers/developers";
import ReportPage from "./pages/report/report";

const DemoPanel = isDemo ? lazy(() => import("./demo/DemoPanel")) : () => null;
const DemoLink = isDemo ? lazy(() => import("./demo/DemoLink")) : () => null;
const Router = isDemo ? HashRouter : BrowserRouter;

function App() {
  const [revision, setRevision] = useState(0);
  return (
    <Router>
      {isDemo && (
        <Suspense fallback={null}>
          <DemoPanel onReset={() => setRevision((value) => value + 1)} />
        </Suspense>
      )}
      <ScrollToAnchor />
      <Routes key={revision}>
        <Route path="/" element={<Home />} />
        <Route path="/stats" element={<Stats />} />
        <Route path="/register" element={<SignUpPage />} />
        <Route path="/privacy" element={<PrivacyPage />} />
        <Route path="/terms" element={<TermsPage />} />
        <Route path="/status" element={<StatusPage />} />
        <Route path="/developers" element={<DevelopersPage />} />
        <Route path="/report" element={<ReportPage />} />
        {isDemo && (
          <Route
            path="/s/:code"
            element={
              <Suspense fallback={null}>
                <DemoLink />
              </Suspense>
            }
          />
        )}
        {isDemo && <Route path="*" element={<Home />} />}
      </Routes>
    </Router>
  );
}

export default App;
