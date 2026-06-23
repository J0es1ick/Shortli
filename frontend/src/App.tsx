import "./App.css";
import Home from "./pages/home/home";
import { BrowserRouter, Route, Routes } from "react-router-dom";
import Stats from "./pages/stats/stats";
import SignUpPage from "./pages/signUp/signUp";

function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<Home />} />
        <Route path="/stats" element={<Stats />} />
        <Route path="/register" element={<SignUpPage />} />
      </Routes>
    </BrowserRouter>
  );
}

export default App;
