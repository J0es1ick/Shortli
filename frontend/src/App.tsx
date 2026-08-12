import "./App.css";
import Home from "./pages/home/home";
import { BrowserRouter, Route, Routes } from "react-router-dom";
import Stats from "./pages/stats/stats";
import SignUpPage from "./pages/signUp/signUp";
import PrivacyPage from "./pages/info/privacy";
import TermsPage from "./pages/info/terms";
import StatusPage from "./pages/info/status";
import DevelopersPage from "./pages/developers/developers";
import ReportPage from "./pages/report/report";

function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<Home />} />
        <Route path="/stats" element={<Stats />} />
        <Route path="/register" element={<SignUpPage />} />
        <Route path="/privacy" element={<PrivacyPage />} />
        <Route path="/terms" element={<TermsPage />} />
        <Route path="/status" element={<StatusPage />} />
        <Route path="/developers" element={<DevelopersPage />} />
        <Route path="/report" element={<ReportPage />} />
      </Routes>
    </BrowserRouter>
  );
}

export default App;
